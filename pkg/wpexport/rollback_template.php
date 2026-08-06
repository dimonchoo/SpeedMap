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
	if ( ! $matches ) {
		WP_CLI::error( 'No speedmap-webp-backup-*.json found in uploads. Run apply first.' );
	}
	rsort( $matches );
	$backup_path = $matches[0];
}

$raw = file_get_contents( $backup_path );
$backup = json_decode( $raw, true );
if ( ! is_array( $backup ) || empty( $backup['items'] ) ) {
	WP_CLI::error( 'Invalid backup: ' . $backup_path );
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

$ok = 0;
$fail = 0;
foreach ( $backup['items'] as $item ) {
	$att_id = isset( $item['attachmentId'] ) ? (int) $item['attachmentId'] : 0;
	if ( ! $att_id ) {
		$fail++;
		continue;
	}

	$old_file = isset( $item['oldAttachedFile'] ) ? $item['oldAttachedFile'] : '';
	$old_mime = isset( $item['oldMime'] ) ? $item['oldMime'] : '';
	$old_url  = isset( $item['oldUrl'] ) ? $item['oldUrl'] : '';
	$new_url  = isset( $item['newUrl'] ) ? $item['newUrl'] : '';

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

	// Reverse URL replace: webp → original
	if ( $new_url && $old_url ) {
		speedmap_rollback_replace_urls( $new_url, $old_url );
		$new_path = wp_parse_url( $new_url, PHP_URL_PATH );
		$old_path = wp_parse_url( $old_url, PHP_URL_PATH );
		if ( $new_path && $old_path && $new_path !== $old_path ) {
			speedmap_rollback_replace_urls( $new_path, $old_path );
		}
	}

	$ok++;
	WP_CLI::log( sprintf( '[restored] id=%d → %s', $att_id, $old_file ) );
}

WP_CLI::success( sprintf( 'Rollback done. restored=%d skipped=%d backup=%s', $ok, $fail, $backup_path ) );
