<?php
/**
 * SpeedMap WebP apply script (WP-CLI eval-file) — DB only
 *
 * WebP files are written by SpeedMap into wp-content/uploads before you run this.
 * Place this PHP next to the WP root (or anywhere), then:
 *   wp eval-file this-file.php --path={{WORDPRESS_PATH}}
 *
 * What it does:
 *   1) Reads embedded manifest (webpRel + old URLs + pages from SpeedMap scan)
 *   2) Verifies each .webp already exists under uploads
 *   3) Writes backup JSON (old meta/URLs) BEFORE mutating
 *   4) Updates attachment file pointer + mime (keeps old files on disk; keeps title/alt/caption)
 *   5) URL search-replace
 *   6) Writes JSON report + QA HTML checklist
 *
 * Rollback: wp eval-file speedmap-webp-rollback-….php --path=…
 *
 * Requires: WP-CLI. Does NOT download or convert images. Does NOT delete originals.
 */
if ( ! defined( 'ABSPATH' ) ) {
	fwrite( STDERR, "Run via: wp eval-file this-file.php --path={{WORDPRESS_PATH}}\n" );
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

$wp_path = isset( $SPEEDMAP_MANIFEST['wordpressPath'] ) ? $SPEEDMAP_MANIFEST['wordpressPath'] : '{{WORDPRESS_PATH}}';

$uploads = wp_upload_dir();
if ( ! empty( $uploads['error'] ) ) {
	WP_CLI::error( 'uploads dir error: ' . $uploads['error'] );
}

$report = array(
	'domain'         => isset( $SPEEDMAP_MANIFEST['domain'] ) ? $SPEEDMAP_MANIFEST['domain'] : '',
	'wordpressPath'  => $wp_path,
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
	if ( count( $rows ) === 1 ) {
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
		// Also match when attached file is already webp with same stem
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
		return 0; // ambiguous
	}
	$guid_like = '%' . $wpdb->esc_like( $basename );
	$guid_ids  = $wpdb->get_col(
		$wpdb->prepare(
			"SELECT ID FROM {$wpdb->posts} WHERE post_type = 'attachment' AND guid LIKE %s LIMIT 3",
			$guid_like
		)
	);
	if ( count( $guid_ids ) === 1 ) {
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

function speedmap_resolve_item( $item, $uploads ) {
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
		'destAbs'      => '',
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

	if ( ! file_exists( $dest_abs ) ) {
		$row['reason'] = 'webp missing on disk (export from SpeedMap first): ' . $webp_rel;
		return $row;
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
	$row['reason'] = '';
	return $row;
}

WP_CLI::log( sprintf( 'SpeedMap WebP DB apply: %d images, path=%s', count( $SPEEDMAP_MANIFEST['images'] ), $wp_path ) );

// Pass 1: resolve + snapshot backup (before any mutation)
$resolved = array();
$backup   = array(
	'domain'        => isset( $SPEEDMAP_MANIFEST['domain'] ) ? $SPEEDMAP_MANIFEST['domain'] : '',
	'wordpressPath' => $wp_path,
	'createdAt'     => gmdate( 'c' ),
	'items'         => array(),
);

foreach ( $SPEEDMAP_MANIFEST['images'] as $item ) {
	$row = speedmap_resolve_item( $item, $uploads );
	$resolved[] = $row;
	if ( $row['status'] !== 'pending' || ! $row['attachmentId'] ) {
		continue;
	}
	$att_id = $row['attachmentId'];
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

$stamp = gmdate( 'Ymd-His' );
$backup_path = trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-backup-' . $stamp . '.json';
file_put_contents( $backup_path, wp_json_encode( $backup, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) );
$report['backup'] = $backup_path;
WP_CLI::log( 'Backup written: ' . $backup_path . ' (' . count( $backup['items'] ) . ' attachments)' );

// Pass 2: mutate (keep old files on disk; preserve title/alt/caption)
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
		// Title / alt / caption intentionally untouched (same attachment).
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
		$row['reason'] = '';
	} else {
		$row['status'] = 'file_only';
		$row['reason'] = 'webp on disk but no unique attachment match (old files kept)';
	}

	unset( $row['destAbs'] );
	$report['items'][] = $row;
	WP_CLI::log( sprintf( '[%s] %s → %s', $row['status'], $row['oldUrl'], $row['newUrl'] ? $row['newUrl'] : $webp_rel ) );
}

$report_path = trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-report-' . $stamp . '.json';
file_put_contents( $report_path, wp_json_encode( $report, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) );

$qa_path = trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-qa-' . $stamp . '.html';
file_put_contents( $qa_path, speedmap_build_qa_html( $report ) );
$report['qaReport'] = $qa_path;
$report['jsonReport'] = $report_path;
file_put_contents( $report_path, wp_json_encode( $report, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES ) );

$ok = 0;
$fail = 0;
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
	$rows = '';
	foreach ( $report['items'] as $it ) {
		$pages_html = '';
		if ( ! empty( $it['pages'] ) && is_array( $it['pages'] ) ) {
			$links = array();
			foreach ( $it['pages'] as $pg ) {
				$links[] = '<a href="' . $esc( $pg ) . '" target="_blank" rel="noopener">' . $esc( $pg ) . '</a>';
			}
			$pages_html = implode( '<br>', $links );
		}
		$status = isset( $it['status'] ) ? $it['status'] : '';
		$rows .= '<tr class="st-' . $esc( $status ) . '">'
			. '<td>' . $esc( $status ) . '</td>'
			. '<td><a href="' . $esc( isset( $it['oldUrl'] ) ? $it['oldUrl'] : '' ) . '" target="_blank" rel="noopener">' . $esc( isset( $it['oldUrl'] ) ? $it['oldUrl'] : '' ) . '</a></td>'
			. '<td><a href="' . $esc( isset( $it['newUrl'] ) ? $it['newUrl'] : '' ) . '" target="_blank" rel="noopener">' . $esc( isset( $it['newUrl'] ) ? $it['newUrl'] : '' ) . '</a></td>'
			. '<td>' . $pages_html . '</td>'
			. '<td>' . $esc( isset( $it['reason'] ) ? $it['reason'] : '' ) . '</td>'
			. '</tr>';
	}

	$title = 'SpeedMap WebP QA — ' . $esc( isset( $report['domain'] ) ? $report['domain'] : '' );
	return '<!DOCTYPE html><html><head><meta charset="utf-8"><title>' . $title . '</title>'
		. '<style>body{font-family:system-ui,sans-serif;margin:24px;color:#111}table{border-collapse:collapse;width:100%}th,td{border:1px solid #ccc;padding:8px;vertical-align:top;font-size:13px}th{background:#f4f4f4;text-align:left}.st-applied{background:#e8f8e8}.st-file_only{background:#fff8e0}.st-failed{background:#fde8e8}a{word-break:break-all}</style>'
		. '</head><body>'
		. '<h1>' . $title . '</h1>'
		. '<p>Applied at: ' . $esc( isset( $report['appliedAt'] ) ? $report['appliedAt'] : '' )
		. ' · WP path: <code>' . $esc( isset( $report['wordpressPath'] ) ? $report['wordpressPath'] : '' ) . '</code>'
		. ' · Quality: ' . $esc( isset( $report['quality'] ) ? $report['quality'] : '' )
		. ' · Backup: <code>' . $esc( isset( $report['backup'] ) ? $report['backup'] : '' ) . '</code></p>'
		. '<p>Open each <strong>Pages</strong> URL and confirm the image loads as WebP (new URL). Compare old vs new. Old files remain on disk for rollback.</p>'
		. '<table><thead><tr><th>Status</th><th>Old URL</th><th>New URL</th><th>Pages (spot-check)</th><th>Notes</th></tr></thead><tbody>'
		. $rows
		. '</tbody></table></body></html>';
}
