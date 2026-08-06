<?php
/**
 * SpeedMap WebP apply script (WP-CLI eval-file)
 *
 * Place this file on the environment (repo / deploy — SpeedMap does not deploy),
 * then run:
 *   wp eval-file this-file.php --path={{WORDPRESS_PATH}}
 *
 * What it does:
 *   1) Reads embedded manifest (prod image URLs + page URLs from SpeedMap scan)
 *   2) Downloads each image from production
 *   3) Converts to WebP (Imagick → GD → cwebp)
 *   4) Writes under wp-content/uploads
 *   5) Updates attachment meta + URL search-replace
 *   6) Writes JSON report + QA HTML checklist for testers
 *
 * Requires: WP-CLI, outbound HTTPS to image hosts, Imagick/GD WebP or cwebp.
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
	}
	if ( count( $rows ) > 1 ) {
		return 0; // ambiguous
	}
	// GUID fallback
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

function speedmap_convert_to_webp( $src_path, $dest_path, $quality ) {
	if ( class_exists( 'Imagick' ) ) {
		try {
			$img = new Imagick( $src_path );
			if ( method_exists( $img, 'setImageFormat' ) ) {
				$img->setImageFormat( 'webp' );
			}
			if ( method_exists( $img, 'setImageCompressionQuality' ) ) {
				$img->setImageCompressionQuality( $quality );
			}
			$ok = $img->writeImage( $dest_path );
			$img->clear();
			$img->destroy();
			if ( $ok && file_exists( $dest_path ) ) {
				return true;
			}
		} catch ( Exception $e ) {
			// fall through
		}
	}

	$info = @getimagesize( $src_path );
	if ( $info && function_exists( 'imagewebp' ) ) {
		$im = null;
		switch ( $info[2] ) {
			case IMAGETYPE_JPEG:
				$im = imagecreatefromjpeg( $src_path );
				break;
			case IMAGETYPE_PNG:
				$im = imagecreatefrompng( $src_path );
				if ( $im ) {
					imagepalettetotruecolor( $im );
					imagealphablending( $im, true );
					imagesavealpha( $im, true );
				}
				break;
			case IMAGETYPE_GIF:
				$im = imagecreatefromgif( $src_path );
				break;
			case IMAGETYPE_WEBP:
				if ( function_exists( 'imagecreatefromwebp' ) ) {
					$im = imagecreatefromwebp( $src_path );
				}
				break;
		}
		if ( $im ) {
			$ok = imagewebp( $im, $dest_path, $quality );
			imagedestroy( $im );
			if ( $ok && file_exists( $dest_path ) ) {
				return true;
			}
		}
	}

	$cwebp = trim( (string) shell_exec( 'command -v cwebp 2>/dev/null' ) );
	if ( $cwebp !== '' ) {
		$cmd = escapeshellcmd( $cwebp ) . ' -q ' . (int) $quality . ' ' . escapeshellarg( $src_path ) . ' -o ' . escapeshellarg( $dest_path ) . ' 2>/dev/null';
		exec( $cmd, $out, $code );
		if ( $code === 0 && file_exists( $dest_path ) ) {
			return true;
		}
	}

	return false;
}

function speedmap_replace_urls( $old_url, $new_url ) {
	global $wpdb;
	if ( ! $old_url || ! $new_url || $old_url === $new_url ) {
		return 0;
	}
	$n = 0;
	// posts
	$n += (int) $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->posts} SET post_content = REPLACE(post_content, %s, %s)", $old_url, $new_url ) );
	$n += (int) $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->posts} SET guid = REPLACE(guid, %s, %s)", $old_url, $new_url ) );
	// postmeta (may break some serialized blobs; prefer exact URL strings)
	$n += (int) $wpdb->query( $wpdb->prepare( "UPDATE {$wpdb->postmeta} SET meta_value = REPLACE(meta_value, %s, %s)", $old_url, $new_url ) );
	return $n;
}

WP_CLI::log( sprintf( 'SpeedMap WebP apply: %d images, quality=%d, path=%s', count( $SPEEDMAP_MANIFEST['images'] ), $quality, $wp_path ) );

foreach ( $SPEEDMAP_MANIFEST['images'] as $item ) {
	$row = array(
		'sourceUrl'    => isset( $item['sourceUrl'] ) ? $item['sourceUrl'] : '',
		'pages'        => isset( $item['pages'] ) && is_array( $item['pages'] ) ? $item['pages'] : array(),
		'status'       => 'failed',
		'reason'       => '',
		'attachmentId' => 0,
		'oldUrl'       => '',
		'newUrl'       => '',
		'webpRel'      => '',
	);

	$source = $row['sourceUrl'];
	if ( ! $source ) {
		$row['reason'] = 'empty sourceUrl';
		$report['items'][] = $row;
		continue;
	}

	$path_hint = isset( $item['pathHint'] ) ? $item['pathHint'] : speedmap_path_hint_from_url( $source );
	$basename  = isset( $item['basename'] ) ? $item['basename'] : basename( parse_url( $source, PHP_URL_PATH ) );
	$basename  = speedmap_strip_size_suffix( $basename );
	$name_no_ext = preg_replace( '/\.[a-zA-Z0-9]+$/', '', $basename );
	$webp_name   = $name_no_ext . '.webp';

	$att_id = speedmap_find_attachment_id( $basename, $path_hint );
	$row['attachmentId'] = $att_id;

	if ( ! $att_id ) {
		$row['reason'] = 'attachment not found (will still try file write)';
	}

	$tmp = download_url( $source, 60 );
	if ( is_wp_error( $tmp ) ) {
		$row['reason'] = 'download failed: ' . $tmp->get_error_message();
		$report['items'][] = $row;
		WP_CLI::warning( $source . ' — ' . $row['reason'] );
		continue;
	}

	$rel_dir = '';
	if ( $att_id ) {
		$attached = get_post_meta( $att_id, '_wp_attached_file', true );
		if ( $attached ) {
			$rel_dir = trailingslashit( dirname( $attached ) );
			if ( $rel_dir === './' ) {
				$rel_dir = '';
			}
		}
	}
	if ( $rel_dir === '' && $path_hint ) {
		$rel_dir = trailingslashit( dirname( $path_hint ) );
		if ( $rel_dir === './' ) {
			$rel_dir = '';
		}
	}
	if ( $rel_dir === '' || $rel_dir === '/' ) {
		$rel_dir = trailingslashit( gmdate( 'Y/m' ) );
	}

	$dest_rel  = $rel_dir . $webp_name;
	$dest_abs  = trailingslashit( $uploads['basedir'] ) . $dest_rel;
	$dest_dir  = dirname( $dest_abs );
	if ( ! wp_mkdir_p( $dest_dir ) ) {
		@unlink( $tmp );
		$row['reason'] = 'cannot create dir ' . $dest_dir;
		$report['items'][] = $row;
		continue;
	}

	$ok = speedmap_convert_to_webp( $tmp, $dest_abs, $quality );
	@unlink( $tmp );
	if ( ! $ok ) {
		$row['reason'] = 'webp convert failed (need Imagick/GD WebP or cwebp)';
		$report['items'][] = $row;
		WP_CLI::warning( $source . ' — ' . $row['reason'] );
		continue;
	}

	$row['webpRel'] = $dest_rel;
	$new_url = trailingslashit( $uploads['baseurl'] ) . str_replace( '\\', '/', $dest_rel );
	$row['newUrl'] = $new_url;
	$row['oldUrl'] = $source;

	if ( $att_id ) {
		$old_url = wp_get_attachment_url( $att_id );
		if ( $old_url ) {
			$row['oldUrl'] = $old_url;
		}
		update_post_meta( $att_id, '_wp_attached_file', $dest_rel );
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
		$row['reason'] = 'wrote webp but no unique attachment match';
	}

	$report['items'][] = $row;
	WP_CLI::log( sprintf( '[%s] %s → %s', $row['status'], $row['oldUrl'], $row['newUrl'] ? $row['newUrl'] : $dest_rel ) );
}

$stamp = gmdate( 'Ymd-His' );
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

WP_CLI::success( sprintf( 'Done. applied=%d file_only=%d failed=%d json=%s qa=%s', $ok, $file_only, $fail, $report_path, $qa_path ) );
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
		. ' · Quality: ' . $esc( isset( $report['quality'] ) ? $report['quality'] : '' ) . '</p>'
		. '<p>Open each <strong>Pages</strong> URL and confirm the image loads as WebP (new URL). Compare old vs new.</p>'
		. '<table><thead><tr><th>Status</th><th>Old URL</th><th>New URL</th><th>Pages (spot-check)</th><th>Notes</th></tr></thead><tbody>'
		. $rows
		. '</tbody></table></body></html>';
}
