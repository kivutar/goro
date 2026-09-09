# Legacy mail (2008)

Goro implements the original mailbox, not RODEX. The server opens it through
an NPC's `openmail` command (or `@mail` on a server that permits the command).
There is no always-available modern mail button.

## Player flow

- **Inbox:** refresh, select a message and press Read, or double-click it.
  Unread messages are blue. The legacy inbox holds 30 messages, displayed seven
  at a time.
- **Write:** enter a character name, subject, and multiline message. A message
  can carry one item stack and Zeny. Drag an inventory item onto the attachment
  slot; stacks prompt for an amount. Remove releases the reserved item.
- **Read:** Get takes attachments; Reply pre-fills the sender and subject.
  Return sends the message and its attachments back, if the server allows it.
  Delete requires an attachment-free message. Return and Delete ask for
  confirmation.
- Escape cancels an amount/confirmation prompt first, then closes the read
  window before the mailbox. Closing a draft releases its reservations.
- New mail is announced in the console. It never discards an in-progress draft.
- Validation errors, server failures, and timeouts appear only in the console;
  they do not add a status row or resize either mail window.

Limits are those of the legacy wire format: 23 bytes for the recipient,
39 for the subject, and 199 for the message. Multibyte text is not silently
split or truncated when sending. Eligibility, fees, available inventory space,
weight, Zeny limits, and delivery success remain server decisions.

## Implementation

- `network/mail_packets.go`: the eight legacy requests and nine server packets,
  including list/read parsing, length validation, and acknowledgements.
- `game/mail.go`: request serialization and acknowledgement handling. Mail
  windows express user intent; they do not send packets or change inventory.
- `ui/mail_window.go`: shared windows, tabs, table, buttons, amount prompt,
  confirmations, and item information. The reusable multiline editor in
  `ui/rotheme/text_area.go` supplies the message body.

Legacy attachment acknowledgements are important for inventory accounting:
the successful add-item acknowledgement removes the reserved quantity from
the client inventory. The server returns cancelled or failed attachments through
ordinary item-add packets. A successful send must not delete that stack twice.
Zeny balances are updated by the server's normal status packets.

Send and retrieval replies contain no mail ID. Only one operation is allowed
at a time, including across closing/reopening the mailbox. A timeout reports an
error but does not forget the pending acknowledgement and risk applying it to
a later message. After successful retrieval, an ID-bearing read reply refreshes
the attachment state before enabling the next operation. This also handles
rAthena sending separate legacy success replies for an item and Zeny.

## Verification

Automated checks:

```sh
CGO_ENABLED=0 go test -tags nofakecgo ./network ./game ./ui ./ui/rotheme -run 'Mail|TextArea' -count=1
CGO_ENABLED=0 go test -tags nofakecgo ./...
CGO_ENABLED=0 staticcheck -tags nofakecgo ./...
```

Coverage includes empty/full inboxes, wire lengths and fragmented packets,
read and missing-message replies, item quantities, Zeny acknowledgement ordering,
send/delete/return failures, duplicate clicks, delayed retrieval replies,
timeouts, closing/reopening with pending replies, draft preservation, UTF-8 byte
limits, multiline editing, Escape, and item-drop routing through overlapping UI.
Published-UI tests cover focus, Tab navigation, captured text selection, and
delete confirmations; world-level tests ensure Enter does not also activate
chat and map transitions retain unprocessed mail acknowledgements.
Idle mailbox, reader, and composer updates allocate nothing and retain their
published content. Editor regressions cover soft-wrap caret placement and
screen-space redraw damage.
Offline renders of the inbox, read window, and composer have also been inspected
with OldRO assets.

Live acceptance checks, still pending:

- [ ] Send plain text to another character; check unread/read and Reply.
- [ ] Send a partial item stack plus Zeny, then retrieve both exactly once.
- [ ] Attach/remove/cancel/close; verify both characters' inventory and Zeny
  before and after relogging.
- [ ] Return an attached message and delete an empty one; verify server storage.
- [ ] Reject a nonexistent recipient, overweight retrieval, and full inventory.
- [ ] Disconnect/reconnect with a request pending; verify no stale draft or
  incorrectly retained inventory reservation.

## Local server compatibility findings

The local rAthena source at `57c5b4d229291e7737c79f80b7b5fd326e00d4b7`
has legacy-path defects that need resolving before live attachment acceptance:

- `clif_parse_Mail_setattach` applies `server_index()` to index zero before
  testing for a Zeny attachment. The unsigned result is outside the inventory,
  so a valid legacy Zeny request is silently rejected.
- `clif_parse_Mail_winopen` and `clif_parse_Mail_refreshinbox` pass the internal
  item index to `mail_removeitem`, which expects the wire index (internal + 2).
  Resetting a draft therefore does not reliably return its reservation.
- `mail_removeitem`'s delivery branch does not clear its reservation entry.
  The send path and subsequent cleanup need live validation for repeated sends.

These are server-side issues, not alternate client packet formats. Goro does
not fabricate inventory refunds or use RODEX packets to hide them.
