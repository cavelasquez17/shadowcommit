# Mood Entries: Privacy and Consent

## Status

Mood entries and photo uploads are not implemented yet. ShadowCommit does not currently collect
mood-entry photos. This document sets the privacy requirements for that feature; uploads must not be
enabled until the consent, access-control, storage, and deletion safeguards below are implemented and
tested.

## What a mood entry contains

When the feature is available, an entry may contain a photo, a feeling label, an optional message, a
creation time, and a visibility setting. The photo and its entry will be linked so deleting the entry
also deletes the photo from active storage.

Photos will only be collected to display the mood entry chosen by the member. They will not be analyzed
to identify people, infer sensitive traits, train models, or target advertising. ShadowCommit will not
send photos to third-party image, face-analysis, or image-hosting services.

## Consent and visibility

Before a member's first upload, the interface must explain what is stored, who can view it, and how to
delete it, and ask for explicit consent. Consent to upload does not make an entry public or shared.
The first-upload confirmation must include this information in plain language:

> ShadowCommit stores your photo with this mood entry on its own infrastructure. New entries are
> private. If you choose to share an entry, signed-in ShadowCommit members can view it. Photos are not
> sent to third-party image or face-analysis services. You can delete an entry and its photo, or delete
> all mood entries and photos when you delete your account. Existing backup copies follow the retention
> limits described in the privacy notice.

Every entry will start **private** and be visible only to its owner. A separate, explicit per-entry
choice may share an entry with signed-in ShadowCommit members. Shared entries will not be public web
pages. The interface must make the current visibility clear and allow the owner to make a shared entry
private again or delete it. Once something has been shared, other members may have seen or copied it;
changing visibility cannot retrieve those copies.

## Storage and access

Photos will be stored on infrastructure controlled by ShadowCommit, separately from public static
assets, under opaque server-generated keys in `UPLOAD_DIR` (production default `/data/uploads`).
Upload filenames will not be storage paths. The application will validate and re-encode images, strip
embedded metadata such as GPS coordinates, and serve them only through an authenticated handler that
checks the entry's visibility. Photo responses will be private and not cacheable.

The member who created an entry will be able to view it. Other signed-in members will be able to view
it only after the owner explicitly shares it. Infrastructure operators with access to the host may
access stored files while maintaining the service or responding to a security incident. Moderation
access and report handling must follow the moderation design before shared-photo reporting is enabled.

## Retention and deletion

An entry and its photo will remain in active storage until the owner deletes the entry or deletes all
mood entries. The feature will provide per-entry deletion and `DELETE /api/moods` to delete all mood
entries and photos belonging to the signed-in member. Both operations must remove the database rows
and files, with tests for the database and filesystem effects, including cleanup when one part fails.
ShadowCommit does not currently provide account deletion; if that is added, it must use the same
cleanup path for all mood data.

Production backups will contain a verified database copy and a matching archive of mood photos. The
pair will be created under a shared lock, and both parts will be retained for 14 days. A deletion
removes the entry and photo from active storage; a backup made before that deletion may retain both
until that backup expires. Restores must use a database and photo archive with the same timestamp.
Backup files will not be available through the application. Account-level deletion must remove every
associated row and active photo file; any later account-deletion flow must use the same cleanup path.

## Changes to this notice

This notice describes the planned mood-entry feature, not a feature currently available in ShadowCommit.
It must be updated if the implementation changes what is collected, where it is stored, who can see it,
or how deletion and backups work.
