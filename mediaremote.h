#ifndef MEDIAREMOTE_BRIDGE_H
#define MEDIAREMOTE_BRIDGE_H

#import <Foundation/Foundation.h>
#import <dispatch/dispatch.h>
#include <CoreFoundation/CoreFoundation.h>

typedef enum {
    MRMediaRemoteCommandPlay = 0,
    MRMediaRemoteCommandPause = 1,
    MRMediaRemoteCommandTogglePlayPause = 2,
    MRMediaRemoteCommandNextTrack = 4,
    MRMediaRemoteCommandPreviousTrack = 5
} MRMediaRemoteCommand;

typedef void (^MRMediaRemoteGetNowPlayingInfoBlock)(CFDictionaryRef information);

void MRMediaRemoteSendCommand(MRMediaRemoteCommand command, CFDictionaryRef userInfo);
void MRMediaRemoteGetNowPlayingInfo(dispatch_queue_t queue,
                                    MRMediaRemoteGetNowPlayingInfoBlock block);

/* notification */
void MRMediaRemoteRegisterForNowPlayingNotifications(dispatch_queue_t queue);

extern void mr_on_change();

/* bridge API */

void mr_refresh();
void mr_start_listener();

void mr_play_pause();
void mr_next();
void mr_prev();

const char* mr_title();
const char* mr_artist();
const char* mr_album();

double mr_duration();
double mr_position();

#endif