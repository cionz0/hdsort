//go:build darwin

package gui

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework AppKit -framework Foundation -framework QuickLookThumbnailing
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#import <QuickLookThumbnailing/QLThumbnailGenerator.h>

static int hdsortQuickLook(const char *path, unsigned char **out, int *outLen) {
	@autoreleasepool {
		NSString *s = [NSString stringWithUTF8String:path];
		if (s == nil) {
			return 0;
		}
		NSURL *url = [NSURL fileURLWithPath:s];
		QLThumbnailGenerationRequest *req = [[QLThumbnailGenerationRequest alloc]
			initWithFileAtURL:url
			size:NSMakeSize(320, 240)
			scale:2
			representationTypes:QLThumbnailGenerationRequestRepresentationTypeAll];
		dispatch_semaphore_t sem = dispatch_semaphore_create(0);
		__block NSData *png = nil;
		[[QLThumbnailGenerator sharedGenerator] generateBestRepresentationForRequest:req completionHandler:^(QLThumbnailRepresentation *rep, NSError *error) {
			if (rep != nil && rep.CGImage != nil) {
				NSBitmapImageRep *bitmap = [[NSBitmapImageRep alloc] initWithCGImage:rep.CGImage];
				png = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
			}
			dispatch_semaphore_signal(sem);
		}];
		if (dispatch_semaphore_wait(sem, dispatch_time(DISPATCH_TIME_NOW, 4 * NSEC_PER_SEC)) != 0 || png == nil || png.length == 0) {
			return 0;
		}
		*outLen = (int)png.length;
		*out = malloc((size_t)png.length);
		if (*out == NULL) {
			return 0;
		}
		memcpy(*out, png.bytes, (size_t)png.length);
		return 1;
	}
}
*/
import "C"
import "unsafe"

func systemPreview(path string) ([]byte, bool, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var out *C.uchar
	var n C.int
	if C.hdsortQuickLook(cpath, &out, &n) == 0 || out == nil || n <= 0 {
		return nil, false, nil
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoBytes(unsafe.Pointer(out), n), true, nil
}
