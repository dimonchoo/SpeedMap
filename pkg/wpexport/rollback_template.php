<?php
/**
 * SpeedMap WebP rollback (WP-CLI eval-file)
 *
 * Restores attachment pointers + URL replaces from the latest
 * speedmap-webp-backup-*.json under uploads (written by apply).
 *
 *   wp eval-file this-file.php --path={{WORDPRESS_PATH}}
 *
 * Optional: SPEEDMAP_BACKUP=/full/path/to/backup.json
 */
if ( ! defined( 'ABSPATH' ) ) {
	fwrite( STDERR, "Run via: wp eval-file this-file.php --path={{WORDPRESS_PATH}}\n" );
	exit( 1 );
}

$uploads = wp_upload_dir();
if ( ! empty( $uploads['error'] ) ) {
	WP_CLI::error( 'uploads dir error: ' . $uploads['error'] );
}

$backup_path = getenv( 'SPEEDMAP_BACKUP' );
if ( ! $backup_path ) {
	$matches = glob( trailingslashit( $uploads['basedir'] ) . 'speedmap-webp-backup-*.json' );
	if ( $matches ) {
		rsort( $matches );
		$backup_path = $matches[0];
	}
}

$backup = null;
if ( $backup_path && file_exists( $backup_path ) ) {
	$raw    = file_get_contents( $backup_path );
	$backup = json_decode( $raw, true );
}

// Fallback: If no backup JSON exists in uploads, construct rollback plan directly from manifest.json
if ( ( ! is_array( $backup ) || empty( $backup['items'] ) ) ) {
	$manifest_path = trailingslashit( dirname( __FILE__ ) ) . 'manifest.json';
	if ( file_exists( $manifest_path ) ) {
		$manifest_raw = file_get_contents( $manifest_path );
		$manifest_data = json_decode( $manifest_raw, true );
		if ( is_array( $manifest_data ) && ! empty( $manifest_data['images'] ) ) {
			WP_CLI::log( 'No backup JSON found; using manifest.json for standalone reverse rollback.' );
			$backup_path = $manifest_path;
			$backup = array(
				'domain'     => isset( $manifest_data['domain'] ) ? $manifest_data['domain'] : '',
				'packageDir' => dirname( __FILE__ ),
				'createdAt'  => gmdate( 'c' ),
				'items'      => array(),
			);
			foreach ( $manifest_data['images'] as $m_img ) {
				$path_hint = isset( $m_img['pathHint'] ) ? $m_img['pathHint'] : '';
				$webp_rel  = isset( $m_img['webpRel'] ) ? $m_img['webpRel'] : '';
				$ext       = strtolower( pathinfo( $path_hint, PATHINFO_EXTENSION ) );
				$backup['items'][] = array(
					'attachmentId'    => 0,
					'oldAttachedFile' => $path_hint,
					'oldMime'         => 'image/' . ( $ext === 'jpg' ? 'jpeg' : $ext ),
					'oldUrl'          => isset( $m_img['sourceUrl'] ) ? $m_img['sourceUrl'] : '',
					'newUrl'          => trailingslashit( $uploads['baseurl'] ) . $webp_rel,
					'webpRel'         => $webp_rel,
				);
			}
		}
	}
}

if ( ! is_array( $backup ) || empty( $backup['items'] ) ) {
	WP_CLI::error( 'No speedmap-webp-backup-*.json found in uploads, and no manifest.json in package dir.' );
}

function speedmap_rollback_replace_urls( $old_url, $new_url ) {
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

WP_CLI::log( sprintf( 'SpeedMap WebP rollback from %s (%d items)', $backup_path, count( $backup['items'] ) ) );

$ok   = 0;
$fail = 0;
foreach ( $backup['items'] as $item ) {
	$att_id   = isset( $item['attachmentId'] ) ? (int) $item['attachmentId'] : 0;
	$old_file = isset( $item['oldAttachedFile'] ) ? $item['oldAttachedFile'] : '';
	$old_mime = isset( $item['oldMime'] ) ? $item['oldMime'] : '';
	$old_url  = isset( $item['oldUrl'] ) ? $item['oldUrl'] : '';
	$new_url  = isset( $item['newUrl'] ) ? $item['newUrl'] : '';
	$webp_rel = isset( $item['webpRel'] ) ? $item['webpRel'] : '';

	// Ensure old_file resolves to a genuine original format (.png, .jpg, .gif, .svg), not .webp
	if ( preg_match( '/\.webp$/i', $old_file ) ) {
		$stem_path = preg_replace( '/\.webp$/i', '', $old_file );
		foreach ( array( 'png', 'jpg', 'jpeg', 'gif', 'svg' ) as $cand_ext ) {
			$cand_file = $stem_path . '.' . $cand_ext;
			if ( file_exists( trailingslashit( $uploads['basedir'] ) . $cand_file ) ) {
				$old_file = $cand_file;
				$old_mime = ( $cand_ext === 'svg' ) ? 'image/svg+xml' : ( 'image/' . ( $cand_ext === 'jpg' ? 'jpeg' : $cand_ext ) );
				break;
			}
		}
	}

	// If attachmentId is missing, resolve dynamically by stem or filename
	if ( ! $att_id && ( $old_file || $webp_rel ) ) {
		$basename  = basename( $old_file ? $old_file : $webp_rel );
		$stem      = preg_replace( '/\.[a-zA-Z0-9]+$/', '', $basename );
		$like_orig = '%' . $wpdb->esc_like( $basename );
		$like_webp = '%' . $wpdb->esc_like( $stem . '.webp' );
		$found_ids = $wpdb->get_col(
			$wpdb->prepare(
				"SELECT post_id FROM {$wpdb->postmeta} WHERE meta_key = '_wp_attached_file' AND (meta_value LIKE %s OR meta_value LIKE %s) LIMIT 1",
				$like_orig,
				$like_webp
			)
		);
		if ( ! empty( $found_ids ) ) {
			$att_id = (int) $found_ids[0];
		}
	}

	if ( ! $att_id ) {
		$fail++;
		continue;
	}

	if ( $old_file ) {
		update_post_meta( $att_id, '_wp_attached_file', $old_file );
	}
	if ( $old_mime ) {
		wp_update_post(
			array(
				'ID'             => $att_id,
				'post_mime_type' => $old_mime,
			)
		);
	}

	// Restore .speedmap-orig backup if exists (e.g. overwritten SVGs)
	$dest_abs = '';
	if ( strpos( $webp_rel, 'wp-content/' ) === 0 ) {
		$wp_root  = dirname( $uploads['basedir'], 2 );
		$dest_abs = trailingslashit( $wp_root ) . $webp_rel;
	} elseif ( $webp_rel ) {
		$dest_abs = trailingslashit( $uploads['basedir'] ) . $webp_rel;
	}
	if ( $dest_abs && file_exists( $dest_abs . '.speedmap-orig' ) ) {
		@copy( $dest_abs . '.speedmap-orig', $dest_abs );
		@unlink( $dest_abs . '.speedmap-orig' );
		WP_CLI::log( 'Restored original SVG from backup: ' . $dest_abs );
	}

	$old_abs = $old_file ? trailingslashit( $uploads['basedir'] ) . ltrim( $old_file, '/' ) : '';
	if ( $old_abs && file_exists( $old_abs ) ) {
		require_once ABSPATH . 'wp-admin/includes/image.php';
		$meta = wp_generate_attachment_metadata( $att_id, $old_abs );
		if ( ! empty( $meta ) ) {
			wp_update_attachment_metadata( $att_id, $meta );
		}
	} elseif ( ! empty( $item['oldAttachmentMeta'] ) && is_array( $item['oldAttachmentMeta'] ) ) {
		wp_update_attachment_metadata( $att_id, $item['oldAttachmentMeta'] );
	}

	// 1) Reverse URL replace: webp → original (from backup snapshot)
	if ( $new_url && $old_url && $new_url !== $old_url ) {
		speedmap_rollback_replace_urls( $new_url, $old_url );
		$new_path = wp_parse_url( $new_url, PHP_URL_PATH );
		$old_path = wp_parse_url( $old_url, PHP_URL_PATH );
		if ( $new_path && $old_path && $new_path !== $old_path ) {
			speedmap_rollback_replace_urls( $new_path, $old_path );
		}
	}

	// 2) Reverse local uploads URLs and paths (ensures rollback works seamlessly across staging/local domains)
	if ( $webp_rel && $old_file && $webp_rel !== $old_file ) {
		$local_webp_url = trailingslashit( $uploads['baseurl'] ) . ltrim( $webp_rel, '/' );
		$local_old_url  = trailingslashit( $uploads['baseurl'] ) . ltrim( $old_file, '/' );
		speedmap_rollback_replace_urls( $local_webp_url, $local_old_url );

		$local_webp_path = wp_parse_url( $local_webp_url, PHP_URL_PATH );
		$local_old_path  = wp_parse_url( $local_old_url, PHP_URL_PATH );
		if ( $local_webp_path && $local_old_path && $local_webp_path !== $local_old_path ) {
			speedmap_rollback_replace_urls( $local_webp_path, $local_old_path );
		}
	}

	$ok++;
	WP_CLI::log( sprintf( '[restored] id=%d → %s', $att_id, $old_file ) );
}

WP_CLI::success( sprintf( 'Rollback done. restored=%d skipped=%d source=%s', $ok, $fail, $backup_path ) );
