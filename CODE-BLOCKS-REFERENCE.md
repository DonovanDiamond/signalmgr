# Code Blocks Reference

`(a)` = method on `*Account`. Line numbers are the start line of the first and last item listed, from the working tree.

## api.go — HTTP core (private)
- `API_URL` — api.go:13 — base URL var, default `http://127.0.0.1:8080`
- `APIError` — api.go:17 — exported non-2xx error (status, message, account, challenge_tokens); use errors.As
- `withQuery` — api.go:37 — appends `url.Values` as a query string to a path; unchanged when empty
- `methods` — api.go:44 — HTTP verb to fiber agent constructor map
- `request` — api.go:55 — runs the fiber agent, maps API errors, returns raw body
- `requestJSON[T]` — api.go:81 — `request` + JSON decode into `T`
- `getRaw`, `get[T]`, `post[T]`, `put[T]`, `delete[T]` — api.go:93–121 — typed request helpers (`getRaw` returns raw bytes for binary responses)

## general.go — package-level calls
- `GetAboutResponse`, `GetAbout` — general.go:9–20 — API info
- `GetHealth` — general.go:27 — health check
- `Configuration`, `GetConfiguration`, `PostConfiguration` — general.go:32–44 — global API config
- `GetLinkAccountQRCodeOptions`, `GetLinkAccountQRCode` — general.go:49–56 — device-link QR (PNG bytes)
- `GetLinkAccountURI` — general.go:67 — device-link URI string
- `SendMessageV2_MessageMention`, `SendMessageV2_LinkPreview`, `SendMessageV2`, `SendMessageResponse`, `PostSend` — general.go:74–120 — send message
- `SendMessageV1`, `PostSendV1` — general.go:124–135 — legacy v1 send
- `GetAttachments`, `GetAttachment`, `DeleteAttachment` — general.go:143–157 — attachments

## account.go — per-account calls
- `Account`, `GetAccounts` — account.go:14–21 — account type + list
- `Account_Configuration`, (a) `GetConfiguration`, `PostConfiguration` — account.go:34–44 — account config
- (a) `PostLinkDevice`, `PostRegister`, `PostRegisterVerify`, `PostUnregister`, `PostRateLimitChallenge`, `PutSettings` — account.go:50–108 — registration/settings
- `Account_PostUsernameResponse`, (a) `PostUsername`, `DeleteUsername` — account.go:116–133 — username
- `GroupPermissions`, `Group`, `GetGroupsOptions`, (a) `GetGroups`, `PostCreateGroup`, `GetGroupOptions`, (a) `GetGroup`, `PutGroupSettings`, `DeleteGroup` — account.go:138–215 — groups
- (a) `PostGroupAdmins`, `DeleteGroupAdmins`, `PostBlockGroup`, `PostJoinGroup`, `PostGroupMembers`, `DeleteGroupMembers`, `PostQuitGroup` — account.go:221–265 — group membership
- `MessageResponse`, `GetMessagesOptions`, (a) `GetMessages` — account.go:270–289 — receive (REST, normal/native mode)
- (a) `GetMessagesSocket` — account.go:320 — receive (websocket, json-rpc mode)
- (a) `PutTypingIndicator`, `DeleteTypingIndicator`, `PutProfile` — account.go:356–374 — typing/profile
- `Identity`, (a) `GetIdentities`, `PutTrustIdentity` — account.go:383–398 — identities
- (a) `PostReaction`, `DeleteReaction`, `PostReceipts` — account.go:409–433 — reactions/receipts
- `StickerPack`, (a) `GetStickerPacks`, `PostStickerPack` — account.go:442–458 — stickers
- `Contact_Nickname`, `Contact_Profile`, `Contact`, `GetContactsOptions`, (a) `GetContacts`, `GetContactOptions`, (a) `GetContact`, `GetContactAvatar`, `PutContact`, `PostContactsSync` — account.go:466–541 — contacts
- `SearchResult`, (a) `GetSearch` — account.go:546–552 — check numbers are registered
- (a) `PostPin`, `DeletePin` — account.go:560–570 — registration lock PIN
- `Device`, (a) `GetDevices`, `DeleteDevice`, `DeleteLocalData` — account.go:575–600 — linked devices, local data wipe
- (a) `GetGroupAvatar` — account.go:608 — group avatar (raw bytes)
- (a) `PostGroupPinMessage`, `DeleteGroupPinMessage` — account.go:613–623 — pin/unpin group message
- `Account_PostPollResponse`, (a) `PostPoll`, `DeletePoll`, `PostPollVote` — account.go:631–655 — polls
- `Account_DeleteRemoteMessageResponse`, (a) `DeleteRemoteMessage` — account.go:665–670 — delete sent message for everyone

## signaltypes/types.go
- 43 payload structs for received envelopes (`MessageEnvelope`, `DataMessage`, `SyncMessage`, `CallMessage`, `Payment`, …) — no funcs

## api_test.go
- httptest tests: escaped path segment survives (DisablePathNormalizing), plain-text discarded body OK, *APIError with challenge_tokens

## gateway/
- Signal Gateway web app (nested module) — index in gateway/CODE-BLOCKS-REFERENCE.md
