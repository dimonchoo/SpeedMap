<?php
/**
 * SpeedMap WebP apply script (WP-CLI eval-file)
 *
 * Unpack the SpeedMap package so this PHP sits next to images/:
 *   package/
 *     apply.php          ← this file
 *     images/001/optimized.webp
 *     …
 *
 * Then on the target WP:
 *   wp eval-file /path/to/package/apply.php --path=/var/www/site
 *
 * What it does:
 *   1) Copies images/{id}/optimized.webp → wp-content/uploads/{webpRel}
 *      (does NOT delete originals on disk)
 *   2) Writes backup JSON (old meta/URLs) BEFORE mutating
 *   3) Updates attachment file pointer + mime (keeps title/alt/caption)
 *   4) URL search-replace in posts / postmeta / guid
 *   5) Writes JSON report + QA HTML checklist
 *
 * Rollback: wp eval-file rollback.php --path=…
 *
 * Requires: WP-CLI. Does NOT download or convert images.
 */
if ( ! defined( 'ABSPATH' ) ) {
	fwrite( STDERR, "Run via: wp eval-file this-file.php --path=/path/to/wordpress\n" );
	exit( 1 );
}

$SPEEDMAP_MANIFEST = json_decode( <<<'SPEEDMAP_JSON'
{{SPEEDMAP_MANIFEST_JSON}}
SPEEDMAP_JSON
, true );

if ( ! is_array( $SPEEDMAP_MANIFEST ) || empty( $SPEEDMAP_MANIFEST['images'] ) ) {
	WP_CLI::error( 'SpeedMap manifest missing or empty.' );
}

$quality = isset( $SPEEDMAP_MANIFEST['quality'] ) ? (int) $SPEEDMAP_MANIFEST['quality'] : 80;
if ( $quality < 1 || $quality > 100 ) {
	$quality = 80;
}

$package_dir = dirname( __FILE__ );
// Optional: wp eval-file apply.php /abs/path/to/package --path=…
if ( isset( $args[0] ) && is_string( $args[0] ) && $args[0] !== '' && is_dir( $args[0] ) ) {
	$package_dir = rtrim( $args[0], '/\\' );
}

$uploads = wp_upload_dir();
if ( ! empty( $uploads['error'] ) ) {
	WP_CLI::error( 'uploads dir error: ' . $uploads['error'] );
}

$report = array(
	'domain'         => isset( $SPEEDMAP_MANIFEST['domain'] ) ? $SPEEDMAP_MANIFEST['domain'] : '',
	'packageDir'     => $package_dir,
	'generated'      => isset( $SPEEDMAP_MANIFEST['generated'] ) ? $SPEEDMAP_MANIFEST['generated'] : '',
	'appliedAt'      => gmdate( 'c' ),
	'quality'        => $quality,
	'items'          => array(),
);

/**
 * Strip WP size suffix: hero-1024x768.png → hero.png
 */
function speedmap_strip_size_suffix( $basename ) {
	return preg_replace( '/-\d+x\d+(?=\.[a-zA-Z0-9]+$)/', '', $basename );
}

function speedmap_path_hint_from_url( $url ) {
	$path = wp_parse_url( $url, PHP_URL_PATH );
	if ( ! $path ) {
		return '';
	}
	$marker = '/wp-content/uploads/';
	$pos    = strpos( $path, $marker );
	if ( $pos === false ) {
		return '';
	}
	$rel = substr( $path, $pos + strlen( $marker ) );
	$rel = speedmap_strip_size_suffix( $rel );
	return ltrim( $rel, '/' );
}

function speedmap_find_attachment_id( $basename, $path_hint ) {
	global $wpdb;
	$basename = speedmap_strip_size_suffix( basename( $basename ) );
	$like     = '%' . $wpdb->esc_like( $basename );
	$rows     = $wpdb->get_col(
		$wpdb->prepare(
			"SELECT post_id FROM {$wpdb->postmeta} WHERE meta_key = '_wp_attached_file' AND meta_value LIKE %s",
			$like
		)
	);
	if ( count( $rows ) == 1 ) {
		return (int) $rows[0];
	}
	if ( $path_hint ) {
		$hint_base = speedmap_strip_size_suffix( $path_hint );
		foreach ( $rows as $pid ) {
			$file = get_post_meta( (int) $pid, '_wp_attached_file', true );
			if ( $file && speedmap_strip_size_suffix( $file ) === $hint_base ) {
				return (int) $pid;
			}
		}
		$hint_stem = preg_replace( '/\.[a-zA-Z0-9]+$/', '', $hint_base );
		foreach ( $rows as $pid ) {
			$file = get_post_meta( (int) $pid, '_wp_attached_file', true );
			if ( ! $file ) {
				continue;
			}
			$file_stem = preg_replace( '/\.[a-zA-Z0-9]+$/', '', speedmap_strip_size_suffix( $file ) );
			if ( $file_stem === $hint_stem ) {
				return (int) $pid;
			}
		}
	}
	if ( count( $rows ) > 1 ) {
		return 0;
	}
	$guid_like = '%' . $wpdb->esc_like( $basename );
	$guid_ids  = $wpdb->get_col(
		$wpdb->prepare(
			"SELECT ID FROM {$wpdb->posts} WHERE post_type = 'attachment' AND guid LIKE %s LIMIT 3",
			$guid_like
		)
	);
	if ( count( $guid_ids ) == 1 ) {
		return (int) $guid_ids[0];
	}
	return 0;
}

function speedmap_replace_urls( $old_url, $new_url ) {
	global $wpdb;
	if ( ! $old_url || ! $new_url || $old_url === $new_url ) {
		return 0;
	}
	$n = 0;
	$n += (int) $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->posts} SET post_content = REPLACE(post_content, %s, %s)", $old_url, $new_url ) );
	$n += (int) $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->posts} SET guid = REPLACE(guid, %s, %s)", $old_url, $new_url ) );
	$n += (int) $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->postmeta} SET meta_value = REPLACE(meta_value, %s, %s)", $old_url, $new_url ) );
	return $n;
}

/**
 * Copy package webp into uploads/{webpRel}. Never deletes the original raster.
 */
function speedmap_copy_package_webp( $package_dir, $item, $dest_abs ) {
	$rel = isset( $item['packageWebp'] ) ? ltrim( str_replace( '\\', '/', $item['packageWebp'] ), '/' ) : '';
	if ( $rel === '' && ! empty( $item['id'] ) ) {
		$rel = 'images/' . $item['id'] . '/optimized.webp';
	}
	if ( $rel === '' ) {
		return new WP_Error( 'speedmap_no_package', 'packageWebp missing in manifest' );
	}
	$src = trailingslashit( $package_dir ) . str_replace( '/', DIRECTORY_SEPARATOR, $rel );
	if ( ! file_exists( $src ) ) {
		return new WP_Error( 'speedmap_missing_src', 'package file missing: ' . $rel );
	}
	$dir = dirname( $dest_abs );
	if ( ! is_dir( $dir ) && ! wp_mkdir_p( $dir ) ) {
		return new WP_Error( 'speedmap_mkdir', 'cannot create ' . $dir );
	}
	if ( ! copy( $src, $dest_abs ) ) {
		return new WP_Error( 'speedmap_copy', 'copy failed: ' . $rel . ' → ' . $dest_abs );
	}
	return $rel;
}

function speedmap_resolve_item( $item, $uploads, $package_dir ) {
	$row = array(
		'sourceUrl'    => isset( $item['sourceUrl'] ) ? $item['sourceUrl'] : '',
		'pages'        => isset( $item['pages'] ) && is_array( $item['pages'] ) ? $item['pages'] : array(),
		'status'       => 'failed',
		'reason'       => '',
		'attachmentId' => 0,
		'oldUrl'       => '',
		'newUrl'       => '',
		'webpRel'      => '',
		'pathHint'     => '',
		'basename'     => '',
		'packageWebp'  => '',
		'destAbs'      => '',
		'copied'       => false,
	);

	$source = $row['sourceUrl'];
	if ( ! $source ) {
		$row['reason'] = 'empty sourceUrl';
		return $row;
	}

	$path_hint = isset( $item['pathHint'] ) ? $item['pathHint'] : speedmap_path_hint_from_url( $source );
	$basename  = isset( $item['basename'] ) ? $item['basename'] : basename( parse_url( $source, PHP_URL_PATH ) );
	$basename  = speedmap_strip_size_suffix( $basename );
	$row['pathHint'] = $path_hint;
	$row['basename'] = $basename;

	$webp_rel = isset( $item['webpRel'] ) ? ltrim( str_replace( '\\', '/', $item['webpRel'] ), '/' ) : '';
	if ( $webp_rel === '' ) {
		$name_no_ext = preg_replace( '/\.[a-zA-Z0-9]+$/', '', $basename );
		$rel_dir     = $path_hint ? trailingslashit( dirname( $path_hint ) ) : '';
		if ( $rel_dir === './' || $rel_dir === '/' ) {
			$rel_dir = '';
		}
		$webp_rel = $rel_dir . $name_no_ext . '.webp';
	}

	$dest_abs = trailingslashit( $uploads['basedir'] ) . $webp_rel;
	$row['webpRel'] = $webp_rel;
	$row['destAbs'] = $dest_abs;
	$row['newUrl']  = trailingslashit( $uploads['baseurl'] ) . $webp_rel;
	$row['oldUrl']  = $source;

	$copy = speedmap_copy_package_webp( $package_dir, $item, $dest_abs );
	if ( is_wp_error( $copy ) ) {
		// Allow re-run if webp already sits in uploads from a prior apply.
		if ( ! file_exists( $dest_abs ) ) {
			$row['reason'] = $copy->get_error_message();
			return $row;
		}
		$row['reason'] = 'package copy skipped; existing uploads webp kept';
	} else {
		$row['packageWebp'] = $copy;
		$row['copied']      = true;
	}

	$att_id              = speedmap_find_attachment_id( $basename, $path_hint );
	$row['attachmentId'] = $att_id;
	if ( $att_id ) {
		$old_url = wp_get_attachment_url( $att_id );
		if ( $old_url ) {
			$row['oldUrl'] = $old_url;
		}
	}
	$row['status'] = 'pending';
	if ( $row['reason'] === '' ) {
		$row['reason'] = '';
	}
	return $row;
}

WP_CLI::log( sprintf( 'SpeedMap WebP apply: %d images, package=%s', count( $SPEEDMAP_MANIFEST['images'] ), $package_dir ) );

// Pass 1: copy into uploads + resolve + snapshot backup (before any DB mutation)
$resolved = array();
$backup   = array(
	'domain'     => isset( $SPEEDMAP_MANIFEST['domain'] ) ? $SPEEDMAP_MANIFEST['domain'] : '',
	'packageDir' => $package_dir,
	'createdAt'  => gmdate( 'c' ),
	'items'      => array(),
);

foreach ( $SPEEDMAP_MANIFEST['images'] as $item ) {
	$row        = speedmap_resolve_item( $item, $uploads, $package_dir );
	$resolved[] = $row;
	if ( $row['status'] !== 'pending' || ! $row['attachmentId'] ) {
		continue;
	}
	$att_id            = $row['attachmentId'];
	$backup['items'][] = array(
		'attachmentId'      => $att_id,
		'oldAttachedFile'   => get_post_meta( $att_id, '_wp_attached_file', true ),
		'oldMime'           => get_post_mime_type( $att_id ),
		'oldUrl'            => $row['oldUrl'],
		'newUrl'            => $row['newUrl'],
		'webpRel'           => $row['webpRel'],
		'title'             => get_the_title( $att_id ),
		'alt'               => get_post_meta( $att_id, '_wp_attachment_image_alt', true ),
		'caption'           => get_post_field( 'post_excerpt', $att_id ),
		'oldAttachmentMeta' => wp_get_attachment_metadata( $att_id ),
	);
}

$stamp       = gmdate( 'Ymd-His' );
$backup_path = trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-backup-' . $stamp . '.json';
file_put_contents( $backup_path, wp_json_encode( $backup, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) );
$report['backup'] = $backup_path;
WP_CLI::log( 'Backup written: ' . $backup_path . ' (' . count( $backup['items'] ) . ' attachments)' );

// Pass 2: mutate DB (keep old raster files on disk; preserve title/alt/caption)
foreach ( $resolved as $row ) {
	if ( $row['status'] !== 'pending' ) {
		$report['items'][] = $row;
		WP_CLI::warning( $row['sourceUrl'] . ' — ' . $row['reason'] );
		continue;
	}

	$att_id   = $row['attachmentId'];
	$dest_abs = $row['destAbs'];
	$webp_rel = $row['webpRel'];
	$new_url  = $row['newUrl'];

	if ( $att_id ) {
		update_post_meta( $att_id, '_wp_attached_file', $webp_rel );
		wp_update_post(
			array(
				'ID'             => $att_id,
				'post_mime_type' => 'image/webp',
			)
		);

		require_once ABSPATH . 'wp-admin/includes/image.php';
		$meta = wp_generate_attachment_metadata( $att_id, $dest_abs );
		if ( ! empty( $meta ) ) {
			wp_update_attachment_metadata( $att_id, $meta );
		}

		if ( ! empty( $row['oldUrl'] ) ) {
			speedmap_replace_urls( $row['oldUrl'], $new_url );
			$old_path = wp_parse_url( $row['oldUrl'], PHP_URL_PATH );
			$new_path = wp_parse_url( $new_url, PHP_URL_PATH );
			if ( $old_path && $new_path && $old_path !== $new_path ) {
				speedmap_replace_urls( $old_path, $new_path );
			}
		}
		$row['status'] = 'applied';
		$row['reason'] = $row['copied'] ? 'copied + DB' : 'DB (webp already in uploads)';
	} else {
		$row['status'] = 'file_only';
		$row['reason'] = 'webp copied/present but no unique attachment match (old files kept)';
	}

	unset( $row['destAbs'] );
	$report['items'][] = $row;
	WP_CLI::log( sprintf( '[%s] %s → %s', $row['status'], $row['oldUrl'], $row['newUrl'] ? $row['newUrl'] : $webp_rel ) );
}

$report_path = trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-report-' . $stamp . '.json';
file_put_contents( $report_path, wp_json_encode( $report, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) );

$qa_path = trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-qa-' . $stamp . '.html';
file_put_contents( $qa_path, speedmap_build_qa_html( $report ) );
$report['qaReport']   = $qa_path;
$report['jsonReport'] = $report_path;
file_put_contents( $report_path, wp_json_encode( $report, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) );

$ok        = 0;
$fail      = 0;
$file_only = 0;
foreach ( $report['items'] as $it ) {
	if ( $it['status'] === 'applied' ) {
		$ok++;
	} elseif ( $it['status'] === 'file_only' ) {
		$file_only++;
	} else {
		$fail++;
	}
}

WP_CLI::success( sprintf( 'Done. applied=%d file_only=%d failed=%d backup=%s json=%s qa=%s', $ok, $file_only, $fail, $backup_path, $report_path, $qa_path ) );
echo wp_json_encode( $report, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) . "\n";

/**
 * QA checklist HTML for testers: status, old→new, pages to spot-check.
 */
function speedmap_build_qa_html( $report ) {
	$esc = function( $s ) {
		return htmlspecialchars( (string) $s, ENT_QUOTES, 'UTF-8' );
	};
	$html  = '<!DOCTYPE html><html><head><meta charset="utf-8"><title>SpeedMap WebP QA</title>';
	$html .= '<style>body{font-family:system-ui,sans-serif;margin:24px;color:#111}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:8px;vertical-align:top;font-size:13px}th{background:#f4f4f4;text-align:left}.st-applied{background:#e8f8e8}.st-file_only{background:#fff8e0}.st-failed{background:#fde8e8}a{word-break:break-all}</style></head><body>';
	$html .= '<h1>SpeedMap WebP QA</h1>';
	$html .= '<p>Applied at: ' . $esc( isset( $report['appliedAt'] ) ? $report['appliedAt'] : '' );
	$html .= ' · Package: <code>' . $esc( isset( $report['packageDir'] ) ? $report['packageDir'] : '' ) . '</code>';
	$html .= ' · Quality: ' . $esc( isset( $report['quality'] ) ? $report['quality'] : '' );
	$html .= ' · Backup: <code>' . $esc( isset( $report['backup'] ) ? $report['backup'] : '' ) . '</code></p>';
	$html .= '<p>Open each <strong>Pages</strong> URL and confirm the image loads as WebP. Old raster files remain on disk for rollback.</p>';
	$html .= '<table><thead><tr><th>Status</th><th>Old URL</th><th>New URL</th><th>Pages</th><th>Notes</th></tr></thead><tbody>';
	foreach ( isset( $report['items'] ) ? $report['items'] : array() as $it ) {
		$st = isset( $it['status'] ) ? $it['status'] : '';
		$html .= '<tr class="st-' . $esc( $st ) . '">';
		$html .= '<td>' . $esc( $st ) . '</td>';
		$html .= '<td>' . $esc( isset( $it['oldUrl'] ) ? $it['oldUrl'] : '' ) . '</td>';
		$html .= '<td>' . $esc( isset( $it['newUrl'] ) ? $it['newUrl'] : '' ) . '</td>';
		$html .= '<td>';
		foreach ( isset( $it['pages'] ) && is_array( $it['pages'] ) ? array_slice( $it['pages'], 0, 5 ) : array() as $p ) {
			$html .= '<div><a href="' . $esc( $p ) . '" target="_blank" rel="noopener">' . $esc( $p ) . '</a></div>';
		}
		$html .= '</td>';
		$html .= '<td>' . $esc( isset( $it['reason'] ) ? $it['reason'] : '' ) . '</td>';
		$html .= '</tr>';
	}
	$html .= '</tbody></table></body></html>';
	return $html;
}
