# Code Blocks Reference

`(a)` = method on `*Account`. Line numbers are as of commit d65a5fb.

## api.go — HTTP core (private)
- `API_URL` — api.go:13 — base URL var, default `http://127.0.0.1:8080`
- `encodeParams` — api.go:21 — query string builder
- `getRaw`, `get[T]`, `post[T]`, `put[T]`, `delete[T]` — api.go:32–98 — typed request helpers
- `completeRequest` — api.go:117 — runs the fiber agent, maps API errors

## general.go — package-level calls
- `GetAbout` / `GetAboutResponse` — general.go:19 — API info
- `GetHealth` — general.go:26 — health check
- `Configuration`, `GetConfiguration`, `PostConfiguration` — general.go:30–42 — global API config
- `GetLinkAccountQRCode` — general.go:48 — device-link QR
- `SendMessageV2`, `SendMessageV2_MessageMention`, `PostSend` — general.go:52–77 — send message
- `GetAttachments`, `GetAttachment`, `DeleteAttachment` — general.go:88–102 — attachments
- `SearchResult`, `GetSearch` — general.go:107–113 — check numbers are registered

## account.go — per-account calls
- `Account`, `GetAccounts` — account.go:12–19 — account type + list
- `Account_Configuration`, (a) `GetConfiguration`, `PostConfiguration` — account.go:32–42 — account config
- (a) `PostLinkDevice`, `PostRegister`, `PostRegisterVerify`, `PostUnregistert`, `PostRateLimitChallenge`, `PutSettings` — account.go:48–106 — registration/settings
- `Account_PostUsernameResponse`, (a) `PostUsername`, `DeleteUsername` — account.go:114–131 — username
- `Group`, (a) `GetGroups`, `PostCreateGroup`, `GetGroup`, `PutGroupSettings`, `DeleteGroup` — account.go:136–188 — groups
- (a) `PostGroupAdmins`, `DeleteGroupAdmins`, `PostBlockGroup`, `PostJoinGroup`, `PostGroupMembers`, `DeleteGroupMembers`, `PostQuitGroup` — account.go:194–238 — group membership
- `MessageResponse`, (a) `GetMessages` — account.go:243–252 — receive (REST, normal/native mode)
- (a) `GetMessagesSocket` — account.go:261 — receive (websocket, json-rpc mode)
- (a) `PutTypingIndicator`, `DeleteTypingIndicator`, `PostProfile` — account.go:293–311 — typing/profile
- `Identity`, (a) `GetIdentities`, `PutTrustIdentity` — account.go:320–334 — identities
- (a) `PostReaction`, `DeleteReaction`, `PostReceipts` — account.go:345–369 — reactions/receipts
- `StickerPack`, (a) `GetStickerPacks`, `PostStickerPack` — account.go:378–394 — stickers
- `Contact`, (a) `GetContacts`, `PostContact`, `PutContactsSync` — account.go:402–431 — contacts

## signaltypes/types.go
- 29 payload structs for received envelopes (`MessageEnvelope`, `DataMessage`, `SyncMessage`, `CallMessage`, `Payment`, …) — no funcs
