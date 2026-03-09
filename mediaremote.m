#import "mediaremote.h"

static NSDictionary *cached;

static void getInfoViaRequest(void) {
    Class cls = NSClassFromString(@"MRNowPlayingRequest");
    if (!cls) {
        NSBundle *bundle = [NSBundle bundleWithPath:@"/System/Library/PrivateFrameworks/MediaRemote.framework"];
        if (bundle) [bundle load];
        cls = NSClassFromString(@"MRNowPlayingRequest");
    }
    if (cls) {
        id item = [cls performSelector:@selector(localNowPlayingItem)];
        if (item) {
            id info = [item performSelector:@selector(nowPlayingInfo)];
            if ([info isKindOfClass:[NSDictionary class]]) {
                cached = [(NSDictionary *)info copy];
            }
        }
    }
}

static void getInfo() {
    getInfoViaRequest();
    if (cached) return;

    dispatch_semaphore_t sem = dispatch_semaphore_create(0);
    MRMediaRemoteGetNowPlayingInfo(dispatch_get_global_queue(DISPATCH_QUEUE_PRIORITY_DEFAULT, 0), ^(CFDictionaryRef i) {
        cached = i ? [(__bridge NSDictionary *)i copy] : nil;
        dispatch_semaphore_signal(sem);
    });
    dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
}

void mr_refresh() {
    getInfo();
}

void mr_start_listener() {

    MRMediaRemoteRegisterForNowPlayingNotifications(dispatch_get_main_queue());

    [[NSNotificationCenter defaultCenter]
        addObserverForName:@"kMRMediaRemoteNowPlayingInfoDidChangeNotification"
        object:nil
        queue:nil
        usingBlock:^(NSNotification *note) {

            getInfo();
            mr_on_change();

        }];
}

static NSString *getStr(NSString *primary, NSString *alt) {
    NSString *s = cached[primary];
    if (!s && alt) s = cached[alt];
    return s ?: @"-";
}

const char* mr_title() {
    NSString *s = getStr(@"kMRMediaRemoteNowPlayingInfoTitle", @"Title");
    return strdup([s UTF8String]);
}

const char* mr_artist() {
    NSString *s = getStr(@"kMRMediaRemoteNowPlayingInfoArtist", @"Artist");
    return strdup([s UTF8String]);
}

const char* mr_album() {
    NSString *s = getStr(@"kMRMediaRemoteNowPlayingInfoAlbum", @"Album");
    return strdup([s UTF8String]);
}

double mr_duration() {
    NSNumber *n = cached[@"kMRMediaRemoteNowPlayingInfoDuration"];
    return n ? [n doubleValue] : 0;
}

double mr_position() {
    NSNumber *n = cached[@"kMRMediaRemoteNowPlayingInfoElapsedTime"];
    return n ? [n doubleValue] : 0;
}

void mr_play_pause() {
    MRMediaRemoteSendCommand(MRMediaRemoteCommandTogglePlayPause, nil);
}

void mr_next() {
    MRMediaRemoteSendCommand(MRMediaRemoteCommandNextTrack, nil);
}

void mr_prev() {
    MRMediaRemoteSendCommand(MRMediaRemoteCommandPreviousTrack, nil);
}