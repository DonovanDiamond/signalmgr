# Code Blocks Reference

`(a)` = method on `*Account`. Line numbers are from the working tree after 0.101 alignment.

## api.go — HTTP core (private)
- `API_URL` — api.go:13 — base URL var, default `http://127.0.0.1:8080`
- `APIError` — api.go:17 — exported non-2xx error (status, message, account, challenge_tokens); use errors.As
- `encodeParams` — api.go:38 — query string builder
- `methods` — api.go:46 — HTTP verb to fiber agent constructor map
- `request` — api.go:57 — runs the fiber agent, maps API errors, returns raw body
- `requestJSON[T]` — api.go:83 — `request` + JSON decode into `T`
- `getRaw`, `get[T]`, `post[T]`, `put[T]`, `delete[T]` — api.go:95–123 — typed request helpers

## general.go — package-level calls
- `GetAboutResponse`, `GetAbout` — general.go:5–16 — API info
- `GetHealth` — general.go:23 — health check
- `Configuration`, `GetConfiguration`, `PostConfiguration` — general.go:28–40 — global API config
- `GetLinkAccountQRCode` — general.go:48 — device-link QR (PNG bytes)
- `SendMessageV2_MessageMention`, `SendMessageV2_LinkPreview`, `SendMessageV2`, `SendMessageResponse`, `PostSend` — general.go:52–98 — send message
- `GetAttachments`, `GetAttachment`, `DeleteAttachment` — general.go:105–119 — attachments

## account.go — per-account calls
- `Account`, `GetAccounts` — account.go:13–20 — account type + list
- `Account_Configuration`, (a) `GetConfiguration`, `PostConfiguration` — account.go:33–43 — account config
- (a) `PostLinkDevice`, `PostRegister`, `PostRegisterVerify`, `PostUnregister`, `PostRateLimitChallenge`, `PutSettings` — account.go:49–107 — registration/settings
- `Account_PostUsernameResponse`, (a) `PostUsername`, `DeleteUsername` — account.go:115–132 — username
- `GroupPermissions`, `Group`, (a) `GetGroups`, `PostCreateGroup`, `GetGroup`, `PutGroupSettings`, `DeleteGroup` — account.go:137–198 — groups
- (a) `PostGroupAdmins`, `DeleteGroupAdmins`, `PostBlockGroup`, `PostJoinGroup`, `PostGroupMembers`, `DeleteGroupMembers`, `PostQuitGroup` — account.go:204–248 — group membership
- `MessageResponse`, (a) `GetMessages` — account.go:253–262 — receive (REST, normal/native mode)
- (a) `GetMessagesSocket` — account.go:271 — receive (websocket, json-rpc mode)
- (a) `PutTypingIndicator`, `DeleteTypingIndicator`, `PutProfile` — account.go:307–325 — typing/profile
- `Identity`, (a) `GetIdentities`, `PutTrustIdentity` — account.go:334–349 — identities
- (a) `PostReaction`, `DeleteReaction`, `PostReceipts` — account.go:360–384 — reactions/receipts
- `StickerPack`, (a) `GetStickerPacks`, `PostStickerPack` — account.go:393–409 — stickers
- `Contact_Nickname`, `Contact_Profile`, `Contact`, (a) `GetContacts`, `PutContact`, `PostContactsSync` — account.go:417–464 — contacts
- `SearchResult`, (a) `GetSearch` — account.go:469–475 — check numbers are registered

## signaltypes/types.go
- 43 payload structs for received envelopes (`MessageEnvelope`, `DataMessage`, `SyncMessage`, `CallMessage`, `Payment`, …) — no funcs
