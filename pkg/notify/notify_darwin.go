//go:build darwin

package notify

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"

@interface SpeedMapNotifyDelegate : NSObject <NSUserNotificationCenterDelegate>
@end

@implementation SpeedMapNotifyDelegate
- (BOOL)userNotificationCenter:(NSUserNotificationCenter *)center shouldPresentNotification:(NSUserNotification *)notification {
    return YES;
}
@end

static SpeedMapNotifyDelegate *smNotifyDelegate = nil;

void showDarwinNotification(const char* title, const char* subtitle, const char* message) {
    @autoreleasepool {
        NSUserNotification *notification = [[NSUserNotification alloc] init];
        if (title && strlen(title) > 0) {
            notification.title = [NSString stringWithUTF8String:title];
        } else {
            notification.title = @"SpeedMap";
        }
        if (subtitle && strlen(subtitle) > 0) {
            notification.subtitle = [NSString stringWithUTF8String:subtitle];
        }
        if (message && strlen(message) > 0) {
            notification.informativeText = [NSString stringWithUTF8String:message];
        }
        notification.soundName = NSUserNotificationDefaultSoundName;

        NSUserNotificationCenter *center = [NSUserNotificationCenter defaultUserNotificationCenter];
        if (!smNotifyDelegate) {
            smNotifyDelegate = [[SpeedMapNotifyDelegate alloc] init];
            [center setDelegate:smNotifyDelegate];
        }
        [center deliverNotification:notification];

        // Gently bounce the Dock icon once if the app is not in the foreground
        if (![NSApp isActive]) {
            [NSApp requestUserAttention:NSInformationalRequest];
        }
    }
}

#pragma clang diagnostic pop
*/
import "C"
import "unsafe"

func sendOSNotification(title, subtitle, message string) {
	cTitle := C.CString(title)
	cSub := C.CString(subtitle)
	cMsg := C.CString(message)
	defer C.free(unsafe.Pointer(cTitle))
	defer C.free(unsafe.Pointer(cSub))
	defer C.free(unsafe.Pointer(cMsg))

	C.showDarwinNotification(cTitle, cSub, cMsg)
}
