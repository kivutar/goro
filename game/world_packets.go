package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	gameui "github.com/kivutar/goro/ui"
)

// handleNetworkPackets drains and applies one frame of world-server packets.
// The boolean result stops the current frame when packet handling changes modes
// or begins a map transition.
func (m *WorldMode) handleNetworkPackets(ctx client.Context, now time.Time) (Mode, bool) {
	for _, pkt := range ctx.Network.DrainPackets() {
		if next, stop := m.handleNetworkPacket(ctx, pkt, now); stop {
			return next, true
		}
	}
	networkErrors := ctx.Network.DrainErrors()
	if handleNetworkDisconnectErrors(ctx, &m.ui.disconnectDialog, networkErrors) {
		return nil, true
	}
	for _, err := range networkErrors {
		glog.Errorf("network frame error: %v", err)
	}

	return nil, false
}

// handleNetworkPacket parses and applies one world-server packet. Its boolean
// result stops the current frame when the packet changes modes or starts a map
// transition.
func (m *WorldMode) handleNetworkPacket(ctx client.Context, pkt network.Packet, now time.Time) (Mode, bool) {
	if handleDisconnectPacket(ctx, &m.ui.disconnectDialog, pkt) {
		return nil, false
	}
	if notify, ok, err := network.ParseMapInfoNotify(pkt); err != nil {
		glog.Errorf("parse map info notification 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyMapInfoNotify(notify)
		return nil, false
	}
	if property, ok, err := network.ParseMapPropertyNotify(pkt); err != nil {
		glog.Errorf("parse map property 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyMapPropertyNotify(ctx, property)
		return nil, false
	}
	if ranking, ok, err := network.ParsePvPRanking(pkt); err != nil {
		glog.Errorf("parse pvp ranking 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPvPRanking(ctx, ranking)
		return nil, false
	}
	if info, ok, err := network.ParsePvPInfo(pkt); err != nil {
		glog.Errorf("parse pvp info 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPvPInfo(info)
		return nil, false
	}
	if hotkeys, ok, err := network.ParseHotkeyList(pkt); err != nil {
		glog.Errorf("parse hotkey list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyHotkeyList(ctx, hotkeys)
		m.ui.shortcutBar.SyncFromSession(ctx)
		return nil, false
	}
	if chat, ok, err := network.ParseChatMessage(pkt); err != nil {
		glog.Errorf("parse chat message 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatMessage(ctx, chat, now)
		return nil, false
	}
	if notify, ok, err := network.ParseExpNotify(pkt); err != nil {
		glog.Errorf("parse exp notify 0x%04X: %v", pkt.ID, err)
	} else if ok {
		addExpNotifyMessage(&m.ui.console, ctx.Resources, notify)
		return nil, false
	}
	if mission, ok, err := network.ParseTaekwonMission(pkt); err != nil {
		glog.Errorf("parse taekwon mission 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyTaekwonMission(ctx, mission)
		return nil, false
	}
	if point, ok, err := network.ParseTaekwonPoint(pkt); err != nil {
		glog.Errorf("parse taekwon point 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("taekwon points current=%d total=%d", point.Point, point.TotalPoint)
		return nil, false
	}
	if ranking, ok, err := network.ParseTaekwonRanking(pkt); err != nil {
		glog.Errorf("parse taekwon ranking 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyTaekwonRanking(ctx, ranking)
		return nil, false
	}
	if place, ok, err := network.ParseStarPlace(pkt); err != nil {
		glog.Errorf("parse star place 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("star gladiator place request=%d", place.Place)
		return nil, false
	}
	if whisper, ok, err := network.ParseWhisperMessage(pkt); err != nil {
		glog.Errorf("parse whisper message 0x%04X: %v", pkt.ID, err)
	} else if ok {
		addWhisperMessage(&m.ui.console, whisper)
		m.addWhisperWindowIncoming(ctx, whisper)
		return nil, false
	}
	if ack, ok, err := network.ParseWhisperAck(pkt); err != nil {
		glog.Errorf("parse whisper ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		addWhisperAck(&m.ui.console, ctx.Resources, ack)
		m.addWhisperWindowAck(ctx, ack)
		return nil, false
	}
	if ack, ok, err := network.ParseWhisperIgnoreAck(pkt); err != nil {
		glog.Errorf("parse whisper ignore ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		addWhisperIgnoreAck(&m.ui.console, ack)
		return nil, false
	}
	if ack, ok, err := network.ParseChatRoomCreateAck(pkt); err != nil {
		glog.Errorf("parse chat room create ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatRoomCreateAck(ctx, ack)
		return nil, false
	}
	if board, ok, err := network.ParseChatRoomBoard(pkt); err != nil {
		glog.Errorf("parse chat room board 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyChatRoomBoard(ctx, board)
		return nil, false
	}
	if destroy, ok, err := network.ParseChatRoomDestroy(pkt); err != nil {
		glog.Errorf("parse chat room destroy 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyChatRoomDestroy(ctx, destroy)
		return nil, false
	}
	if enter, ok, err := network.ParseChatRoomEnter(pkt); err != nil {
		glog.Errorf("parse chat room enter 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatRoomEnter(ctx, enter)
		return nil, false
	}
	if joined, ok, err := network.ParseChatRoomMemberJoin(pkt); err != nil {
		glog.Errorf("parse chat room member join 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatRoomMemberJoin(ctx, joined)
		return nil, false
	}
	if left, ok, err := network.ParseChatRoomMemberLeave(pkt); err != nil {
		glog.Errorf("parse chat room member leave 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatRoomMemberLeave(ctx, left)
		return nil, false
	}
	if change, ok, err := network.ParseChatRoomChange(pkt); err != nil {
		glog.Errorf("parse chat room change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatRoomChange(ctx, change)
		return nil, false
	}
	if role, ok, err := network.ParseChatRoomRoleChange(pkt); err != nil {
		glog.Errorf("parse chat room role change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleChatRoomRoleChange(ctx, role)
		return nil, false
	}
	if emotion, ok, err := network.ParseEmotionNotify(pkt); err != nil {
		glog.Errorf("parse emotion 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyEmotionNotify(ctx, emotion)
		return nil, false
	}
	if change, ok, err := network.ParseMapChange(pkt); err != nil {
		glog.Errorf("parse map change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.startMapFadeOut(change, time.Now())
		return nil, true
	}
	if enter, err := network.ParseMapAcceptEnter(pkt); err == nil {
		applyMapAcceptEnter(ctx, enter)
		sendLessEffectPreference(ctx)
		if m.pendingWarp {
			m.pendingWarp = false
			return m.nextWorldMode(), true
		}
		return nil, false
	}
	if ack, ok, err := network.ParseActorNameAck(pkt); err != nil {
		glog.Errorf("parse actor name ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyActorNameAck(ctx, ack)
		return nil, false
	}
	if ack, ok, err := network.ParseRestartAck(pkt); err != nil {
		glog.Errorf("parse restart ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		refusedCharacterSelect := !ack.Allowed && m.ui.escapeMenu.PendingAction() == gameui.EscapeMenuActionCharacterSelect
		if m.ui.escapeMenu.ApplyRestartAck(ack) {
			m.startCharacterSelectFadeOut(now)
			return nil, true
		}
		if refusedCharacterSelect {
			m.addLeaveWorldRefusalMessage(ctx)
		}
		return nil, false
	}
	if ack, ok, err := network.ParseQuitGameAck(pkt); err != nil {
		glog.Errorf("parse quit game ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyQuitGameAck(ctx, ack)
		return nil, false
	}
	if dialog, ok, err := network.ParseNPCDialog(pkt); err != nil {
		glog.Errorf("parse npc dialog 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.npcDialog.Apply(dialog)
		return nil, false
	}
	if compass, ok, err := network.ParseMinimapCompass(pkt); err != nil {
		glog.Errorf("parse minimap compass 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.minimap.ApplyCompass(compass.ID, compass.Type, compass.X, compass.Y, compass.Color, time.Now())
		return nil, false
	}
	if update, ok, err := network.ParseMapCellUpdate(pkt); err != nil {
		glog.Errorf("parse map cell update 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyMapCellUpdate(ctx, update)
		return nil, false
	}
	if ack, ok, err := network.ParseSelfMoveAck(pkt); err != nil {
		glog.Errorf("parse self move ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applySelfMoveAck(ctx, ack)
		m.clearLocalActorAction(ctx)
		glog.Debugf("walk ack from=%d,%d to=%d,%d tick=%d", ack.FromX, ack.FromY, ack.ToX, ack.ToY, ack.ServerTick)
		m.continuePendingAttack(ctx, "walk ack")
		m.continuePendingPickup(ctx, "walk ack")
		m.skills().ContinuePendingTarget(ctx, "walk ack")
		return nil, false
	}
	if position, ok, err := network.ParseActorSetPosition(pkt); err != nil {
		glog.Errorf("parse actor set position 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if isLocalActor(ctx, position.ID) {
			glog.Debugf("local position fix id=%d x=%d y=%d", position.ID, position.X, position.Y)
		}
		applyActorSetPosition(ctx, position)
		if isLocalActor(ctx, position.ID) {
			m.continuePendingAttack(ctx, "position fix")
			m.continuePendingPickup(ctx, "position fix")
			m.skills().ContinuePendingTarget(ctx, "position fix")
		}
		return nil, false
	}
	if position, ok, err := network.ParseActorJumpPosition(pkt); err != nil {
		glog.Errorf("parse actor jump position 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if isLocalActor(ctx, position.ID) {
			glog.Debugf("local jump position id=%d x=%d y=%d", position.ID, position.X, position.Y)
		}
		applyActorJumpPosition(ctx, position)
		return nil, false
	}
	if item, ok, err := network.ParseFloorItemEntry(pkt); err != nil {
		glog.Errorf("parse floor item entry 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyFloorItemEntry(ctx, item)
		return nil, false
	}
	if disappear, ok, err := network.ParseFloorItemDisappear(pkt); err != nil {
		glog.Errorf("parse floor item disappear 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyFloorItemDisappear(ctx, disappear)
		return nil, false
	}
	if pickup, ok, err := network.ParseItemPickupAck(pkt); err != nil {
		glog.Errorf("parse item pickup ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if item, gained, success := m.applyItemPickupAck(ctx, pickup); success {
			display := pickup
			display.Amount = uint16(gained)
			message := formatPickupConsoleMessage(ctx.Resources, display)
			glog.Debugf("console pickup message item_id=%d amount=%d text=%q", pickup.ItemID, pickup.Amount, message)
			m.ui.console.AddBlueMessage("%s", message)
			m.ui.itemPickup.Show(ctx, item, gained, time.Now())
		} else {
			m.ui.console.AddErrorMessage("Pickup failed item %d result=%d", pickup.ItemID, pickup.Result)
		}
		return nil, false
	}
	if useAck, ok, err := network.ParseUseItemAck(pkt); err != nil {
		glog.Errorf("parse use item ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("use item ack index=%d item=%d aid=%d amount=%d result=%d", useAck.Index, useAck.ItemID, useAck.AID, useAck.Amount, useAck.Result)
		depletedItemID := useAck.ItemID
		if depletedItemID == 0 {
			if item, found := findSessionInventoryItem(ctx.Session, useAck.Index); found {
				depletedItemID = item.ItemID
			}
		}
		m.addItemUseEffect(ctx, useAck)
		applyUseItemAck(ctx, useAck)
		if useAck.Result != 0 && useAck.Amount == 0 && m.ui.shortcutBar.ClearDepletedItem(ctx, useAck.Index, depletedItemID) {
			glog.Debugf("shortcut item depleted index=%d item=%d", useAck.Index, depletedItemID)
		}
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if _, ok, err := network.ParsePetCaptureStart(pkt); err != nil {
		glog.Errorf("parse pet capture start 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.startPetCapture(ctx)
		return nil, false
	}
	if petCapture, ok, err := network.ParsePetCaptureResult(pkt); err != nil {
		glog.Errorf("parse pet capture result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPetCaptureResult(ctx, petCapture)
		return nil, false
	}
	if petProperty, ok, err := network.ParsePetProperty(pkt); err != nil {
		glog.Errorf("parse pet property 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPetProperty(ctx, petProperty)
		return nil, false
	}
	if petFeed, ok, err := network.ParsePetFeedResult(pkt); err != nil {
		glog.Errorf("parse pet feed 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPetFeedResult(ctx, petFeed)
		return nil, false
	}
	if petState, ok, err := network.ParsePetStateChange(pkt); err != nil {
		glog.Errorf("parse pet state 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPetStateChange(ctx, petState)
		return nil, false
	}
	if petEggs, ok, err := network.ParsePetEggList(pkt); err != nil {
		glog.Errorf("parse pet egg list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPetEggList(ctx, petEggs)
		return nil, false
	}
	if petAction, ok, err := network.ParsePetAction(pkt); err != nil {
		glog.Errorf("parse pet action 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyPetAction(ctx, petAction)
		return nil, false
	}
	if identifyList, ok, err := network.ParseItemIdentifyList(pkt); err != nil {
		glog.Errorf("parse item identify list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("item identify list indexes=%v", identifyList.Indexes)
		m.ui.identifyWindow.OpenList(ctx, identifyList)
		return nil, false
	}
	if identifyAck, ok, err := network.ParseItemIdentifyAck(pkt); err != nil {
		glog.Errorf("parse item identify ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("item identify ack index=%d success=%v", identifyAck.Index, identifyAck.Success)
		applyItemIdentifyAck(ctx, identifyAck)
		m.ui.identifyWindow.ApplyAck(ctx, identifyAck)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if arrowList, ok, err := network.ParseMakingArrowList(pkt); err != nil {
		glog.Errorf("parse making arrow list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("making arrow list items=%v", arrowList.ItemIDs)
		m.ui.makingArrow.OpenList(ctx, arrowList)
		return nil, false
	}
	if makingList, ok, err := network.ParseMakableItemList(pkt); err != nil {
		glog.Errorf("parse makable item list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("makable item list items=%v", makingList.Items)
		m.ui.makingItem.OpenList(ctx, makingList)
		return nil, false
	}
	if makingAck, ok, err := network.ParseMakingItemAck(pkt); err != nil {
		glog.Errorf("parse item creation ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("item creation ack item=%d result=%d success=%v alchemist=%v", makingAck.ItemID, makingAck.Result, makingAck.Success(), makingAck.Alchemist())
		m.ui.makingItem.ApplyAck(ctx, makingAck)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if repairList, ok, err := network.ParseRepairItemList(pkt); err != nil {
		glog.Errorf("parse repair item list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("repair item list items=%v", repairList.Items)
		m.ui.repairItem.OpenList(ctx, repairList)
		return nil, false
	}
	if repairAck, ok, err := network.ParseRepairItemAck(pkt); err != nil {
		glog.Errorf("parse repair item ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("repair item ack index=%d success=%v", repairAck.Index, repairAck.Success())
		m.ui.repairItem.ApplyAck(ctx, repairAck)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if refineList, ok, err := network.ParseWeaponRefineList(pkt); err != nil {
		glog.Errorf("parse weapon refine list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("weapon refine list items=%v", refineList.Items)
		m.ui.weaponRefine.OpenList(ctx, refineList)
		return nil, false
	}
	if refineAck, ok, err := network.ParseWeaponRefineAck(pkt); err != nil {
		glog.Errorf("parse weapon refine ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("weapon refine ack item=%d result=%d success=%v", refineAck.ItemID, refineAck.Result, refineAck.Success())
		m.ui.weaponRefine.ApplyAck(ctx, refineAck)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if compositionList, ok, err := network.ParseItemCompositionList(pkt); err != nil {
		glog.Errorf("parse item composition list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("item composition list indexes=%v", compositionList.Indexes)
		m.ui.cardWindow.OpenList(ctx, m.ui.inventoryBag.PendingCardIndex(), compositionList)
		return nil, false
	}
	if compositionAck, ok, err := network.ParseItemCompositionAck(pkt); err != nil {
		glog.Errorf("parse item composition ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("item composition ack equip_index=%d card_index=%d success=%v", compositionAck.EquipIndex, compositionAck.CardIndex, compositionAck.Success)
		applyItemCompositionAck(ctx, compositionAck)
		m.ui.cardWindow.ApplyAck(ctx, compositionAck)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if items, ok, err := network.ParseInventoryItemList(pkt); err != nil {
		glog.Errorf("parse inventory item list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyInventoryItemList(ctx, items)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if itemDelete, ok, err := network.ParseInventoryItemDelete(pkt); err != nil {
		glog.Errorf("parse inventory item delete 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyInventoryItemDelete(ctx, itemDelete)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if equipAck, ok, err := network.ParseInventoryEquipAck(pkt); err != nil {
		glog.Errorf("parse inventory equip ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyInventoryEquipAck(ctx, equipAck)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if arrow, ok, err := network.ParseEquippedArrow(pkt); err != nil {
		glog.Errorf("parse equipped arrow 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("equipped arrow index=%d", arrow.Index)
		applyEquippedArrow(ctx, arrow)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if storageItems, ok, err := network.ParseStorageItemList(pkt); err != nil {
		glog.Errorf("parse storage item list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyStorageItemList(ctx, storageItems)
		m.ui.storageWindow.OpenWindow(ctx)
		return nil, false
	}
	if cartItems, ok, err := network.ParseCartItemList(pkt); err != nil {
		glog.Errorf("parse cart item list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("cart item list items=%d", len(cartItems))
		applyCartItemList(ctx, cartItems)
		m.ui.cartWindow.ClampScroll(ctx.Session)
		m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
		return nil, false
	}
	if storageAmount, ok, err := network.ParseStorageAmount(pkt); err != nil {
		glog.Errorf("parse storage amount 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyStorageAmount(ctx, storageAmount)
		m.ui.storageWindow.OpenWindow(ctx)
		return nil, false
	}
	if cartAmount, ok, err := network.ParseCartAmount(pkt); err != nil {
		glog.Errorf("parse cart amount 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("cart amount count=%d/%d weight=%d/%d", cartAmount.Amount, cartAmount.MaxAmount, cartAmount.Weight, cartAmount.MaxWeight)
		applyCartAmount(ctx, cartAmount)
		m.ui.cartWindow.ClampScroll(ctx.Session)
		m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
		return nil, false
	}
	if friends, ok, err := network.ParseFriendsList(pkt); err != nil {
		glog.Errorf("parse friends list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyFriendsList(ctx, friends)
		return nil, false
	}
	if friendState, ok, err := network.ParseFriendState(pkt); err != nil {
		glog.Errorf("parse friend state 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyFriendState(ctx, friendState)
		return nil, false
	}
	if friendRequest, ok, err := network.ParseFriendRequest(pkt); err != nil {
		glog.Errorf("parse friend request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.openFriendRequest(ctx, friendRequest)
		return nil, false
	}
	if friendAdded, ok, err := network.ParseFriendAddResult(pkt); err != nil {
		glog.Errorf("parse friend add result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyFriendAddResult(ctx, friendAdded)
		m.addFriendResultMessage(friendAdded)
		return nil, false
	}
	if friendDeleted, ok, err := network.ParseFriendDelete(pkt); err != nil {
		glog.Errorf("parse friend delete 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if friend, removed := applyFriendDelete(ctx, friendDeleted); removed {
			name := friend.Name
			if strings.TrimSpace(name) == "" {
				name = "Friend"
			}
			m.ui.console.AddSystemMessage("%s removed from your friend list.", name)
		}
		return nil, false
	}
	if partyCreate, ok, err := network.ParsePartyCreateResult(pkt); err != nil {
		glog.Errorf("parse party create result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handlePartyCreateResult(ctx, partyCreate)
		return nil, false
	}
	if partyList, ok, err := network.ParsePartyList(pkt); err != nil {
		glog.Errorf("parse party list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyList(ctx, partyList)
		return nil, false
	}
	if partyInvite, ok, err := network.ParsePartyInviteRequest(pkt); err != nil {
		glog.Errorf("parse party invite request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.openPartyInviteRequest(ctx, partyInvite)
		return nil, false
	}
	if partyInviteAnswer, ok, err := network.ParsePartyInviteAnswer(pkt); err != nil {
		glog.Errorf("parse party invite answer 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handlePartyInviteAnswer(partyInviteAnswer)
		return nil, false
	}
	if partyOption, ok, err := network.ParsePartyOption(pkt); err != nil {
		glog.Errorf("parse party option 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyOption(ctx, partyOption)
		return nil, false
	}
	if partyConfig, ok, err := network.ParsePartyInviteConfig(pkt); err != nil {
		glog.Errorf("parse party invite config 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyInviteConfig(ctx, partyConfig)
		m.ui.partySettings.Rebind(ctx)
		return nil, false
	}
	if guildBelonging, ok, err := network.ParseGuildBelonging(pkt); err != nil {
		glog.Errorf("parse guild belonging 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildBelonging(ctx, guildBelonging)
		m.requestActorGuildEmblem(ctx, guildBelonging.GuildID, guildBelonging.EmblemVersion)
		return nil, false
	}
	if guildInfo, ok, err := network.ParseGuildInfo(pkt); err != nil {
		glog.Errorf("parse guild info 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildDetails(ctx, guildInfo)
		m.requestActorGuildEmblem(ctx, guildInfo.GuildID, guildInfo.EmblemVersion)
		return nil, false
	}
	if guildMembers, ok, err := network.ParseGuildMembers(pkt); err != nil {
		glog.Errorf("parse guild members 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildMembers(ctx, guildMembers)
		return nil, false
	}
	if guildRelations, ok, err := network.ParseGuildRelations(pkt); err != nil {
		glog.Errorf("parse guild relations 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildRelations(ctx, guildRelations)
		m.ui.guildWindow.Refresh(ctx)
		return nil, false
	}
	if guildMemberState, ok, err := network.ParseGuildMemberState(pkt); err != nil {
		glog.Errorf("parse guild member state 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildMemberState(ctx, guildMemberState)
		if guildMemberState.State == 0 {
			m.ui.minimap.ApplyGuildMemberPosition(guildMemberState.AccountID, -1, -1)
		}
		m.ui.guildWindow.Refresh(ctx)
		return nil, false
	}
	if guildMemberLocation, ok, err := network.ParseGuildMemberLocation(pkt); err != nil {
		glog.Errorf("parse guild member location 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.minimap.ApplyGuildMemberPosition(guildMemberLocation.AccountID, int(guildMemberLocation.X), int(guildMemberLocation.Y))
		return nil, false
	}
	if guildMember, ok, err := network.ParseGuildMemberInfo(pkt); err != nil {
		glog.Errorf("parse guild member info 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildMember(ctx, guildMember)
		return nil, false
	}
	if memberPositions, ok, err := network.ParseGuildMemberPositions(pkt); err != nil {
		glog.Errorf("parse guild member positions 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildMemberPositions(ctx, memberPositions)
		m.ui.guildWindow.Refresh(ctx)
		return nil, false
	}
	if guildPositions, ok, err := network.ParseGuildPositions(pkt); err != nil {
		glog.Errorf("parse guild positions 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildPositions(ctx, guildPositions)
		return nil, false
	}
	if guildPositionNames, ok, err := network.ParseGuildPositionNames(pkt); err != nil {
		glog.Errorf("parse guild position names 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildPositionNames(ctx, guildPositionNames)
		return nil, false
	}
	if guildSkills, ok, err := network.ParseGuildSkillInfo(pkt); err != nil {
		glog.Errorf("parse guild skills 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildSkills(ctx, guildSkills)
		return nil, false
	}
	if guildExpelHistory, ok, err := network.ParseGuildExpelHistory(pkt); err != nil {
		glog.Errorf("parse guild expel history 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildExpelHistory(ctx, guildExpelHistory)
		return nil, false
	}
	if guildNotice, ok, err := network.ParseGuildNotice(pkt); err != nil {
		glog.Errorf("parse guild notice 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleGuildNotice(ctx, guildNotice)
		return nil, false
	}
	if guildEmblem, ok, err := network.ParseGuildEmblemImage(pkt); err != nil {
		glog.Errorf("parse guild emblem 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyGuildEmblemImage(ctx, guildEmblem)
		return nil, false
	}
	if guildEmblemChange, ok, err := network.ParseGuildEmblemChange(pkt); err != nil {
		glog.Errorf("parse guild emblem change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyGuildEmblemChange(ctx, guildEmblemChange)
		return nil, false
	}
	if guildCreate, ok, err := network.ParseGuildCreationResult(pkt); err != nil {
		glog.Errorf("parse guild create result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleGuildCreationResult(ctx, guildCreate)
		return nil, false
	}
	if guildInvite, ok, err := network.ParseGuildInviteRequest(pkt); err != nil {
		glog.Errorf("parse guild invite request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.openGuildInviteRequest(ctx, guildInvite)
		return nil, false
	}
	if guildInviteAck, ok, err := network.ParseGuildInviteAck(pkt); err != nil {
		glog.Errorf("parse guild invite ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleGuildInviteAck(guildInviteAck)
		return nil, false
	}
	if allianceRequest, ok, err := network.ParseGuildAllianceRequest(pkt); err != nil {
		glog.Errorf("parse guild alliance request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.openGuildAllianceRequest(ctx, allianceRequest)
		return nil, false
	}
	if allianceResult, ok, err := network.ParseGuildAllianceResult(pkt); err != nil {
		glog.Errorf("parse guild alliance result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleGuildAllianceResult(allianceResult)
		return nil, false
	}
	if hostilityResult, ok, err := network.ParseGuildHostilityResult(pkt); err != nil {
		glog.Errorf("parse guild hostility result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleGuildHostilityResult(hostilityResult)
		return nil, false
	}
	if relationDeleted, ok, err := network.ParseGuildRelationDeleted(pkt); err != nil {
		glog.Errorf("parse deleted guild relation 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildRelationDeleted(ctx, relationDeleted)
		m.ui.guildWindow.Refresh(ctx)
		return nil, false
	}
	if relationAdded, ok, err := network.ParseGuildRelationAdded(pkt); err != nil {
		glog.Errorf("parse added guild relation 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyLocalGuildRelation(ctx, relationAdded)
		m.ui.guildWindow.Refresh(ctx)
		return nil, false
	}
	if partyMember, ok, err := network.ParsePartyMemberJoin(pkt); err != nil {
		glog.Errorf("parse party member join 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyMemberJoin(ctx, partyMember)
		return nil, false
	}
	if partyLeave, ok, err := network.ParsePartyMemberLeave(pkt); err != nil {
		glog.Errorf("parse party member leave 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if !applyPartyMemberLeave(ctx, partyLeave) {
			m.ui.console.AddErrorMessage("Cannot leave party on this map.")
		}
		return nil, false
	}
	if partyHP, ok, err := network.ParsePartyMemberHP(pkt); err != nil {
		glog.Errorf("parse party member hp 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyMemberHP(ctx, partyHP)
		return nil, false
	}
	if partyPosition, ok, err := network.ParsePartyMemberPosition(pkt); err != nil {
		glog.Errorf("parse party member position 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyMemberPosition(ctx, partyPosition)
		return nil, false
	}
	if partyChat, ok, err := network.ParsePartyChat(pkt); err != nil {
		glog.Errorf("parse party chat 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyPartyChat(ctx, partyChat, &m.ui.console)
		return nil, false
	}
	if tradeRequest, ok, err := network.ParseTradeRequest(pkt); err != nil {
		glog.Errorf("parse trade request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.openTradeRequest(ctx, tradeRequest)
		return nil, false
	}
	if adoptionRequest, ok, err := network.ParseAdoptionRequest(pkt); err != nil {
		glog.Errorf("parse adoption request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.openAdoptionRequest(ctx, adoptionRequest)
		return nil, false
	}
	if adoptionMessage, ok, err := network.ParseAdoptionMessage(pkt); err != nil {
		glog.Errorf("parse adoption message 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleAdoptionMessage(adoptionMessage)
		return nil, false
	}
	if network.ParseAdoptionStarted(pkt) {
		return nil, false
	}
	if showEquip, ok, err := network.ParseShowEquipConfig(pkt); err != nil {
		glog.Errorf("parse show equip config 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if ctx.Session != nil {
			ctx.Session.ShowEquip = showEquip
		}
		return nil, false
	}
	if lessEffects, ok, err := network.ParseLessEffect(pkt); err != nil {
		glog.Errorf("parse less effect 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if ctx.Session != nil {
			ctx.Session.LessEffects = lessEffects
		}
		return nil, false
	}
	if viewedEquip, ok, err := network.ParseViewedEquipment(pkt); err != nil {
		glog.Errorf("parse viewed equipment 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.viewEquipWindow.Open(ctx, viewedEquip, m)
		return nil, false
	}
	if tradeResponse, ok, err := network.ParseTradeResponse(pkt); err != nil {
		glog.Errorf("parse trade response 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleTradeResponse(ctx, tradeResponse)
		return nil, false
	}
	if tradeItem, ok, err := network.ParseTradeItem(pkt); err != nil {
		glog.Errorf("parse trade item 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.tradeWindow.AddReceivedItem(ctx, tradeItem)
		return nil, false
	}
	if tradeAck, ok, err := network.ParseTradeAddItemAck(pkt); err != nil {
		glog.Errorf("parse trade add item ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.tradeWindow.AddOwnItemAck(ctx, tradeAck)
		return nil, false
	}
	if tradeConclude, ok, err := network.ParseTradeConclude(pkt); err != nil {
		glog.Errorf("parse trade conclude 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.tradeWindow.SetConcluded(ctx, tradeConclude.Other)
		return nil, false
	}
	if network.ParseTradeCanceled(pkt) {
		m.ui.tradeWindow.Close(ctx)
		m.ui.console.AddErrorMessage("Trade canceled.")
		return nil, false
	}
	if tradeExec, ok, err := network.ParseTradeExec(pkt); err != nil {
		glog.Errorf("parse trade exec 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.handleTradeExec(ctx, tradeExec)
		return nil, false
	}
	if network.ParseTradeUndo(pkt) {
		m.ui.tradeWindow.Undo(ctx)
		return nil, false
	}
	if storageItem, ok, err := network.ParseStorageItemAdded(pkt); err != nil {
		glog.Errorf("parse storage item added 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyStorageItemAdded(ctx, storageItem)
		m.ui.storageWindow.OpenWindow(ctx)
		m.ui.storageWindow.ClampScroll(ctx.Session)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if cartItem, ok, err := network.ParseCartItemAdded(pkt); err != nil {
		glog.Errorf("parse cart item added 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("cart item added index=%d item=%d amount=%d", cartItem.Index, cartItem.ItemID, cartItem.Amount)
		applyCartItemAdded(ctx, cartItem)
		m.ui.cartWindow.ClampScroll(ctx.Session)
		m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
		m.ui.inventoryBag.ClampScroll(ctx.Session)
		return nil, false
	}
	if ack, ok, err := network.ParseCartAddAck(pkt); err != nil {
		glog.Errorf("parse cart add ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		switch ack.Result {
		case 0:
			m.ui.console.AddErrorMessage("Cart is overweight.")
			glog.Warnf("cart add rejected result=%d reason=weight", ack.Result)
		case 1:
			m.ui.console.AddErrorMessage("Cart has too many items.")
			glog.Warnf("cart add rejected result=%d reason=count", ack.Result)
		default:
			m.ui.console.AddErrorMessage("Cart add failed.")
			glog.Warnf("cart add rejected result=%d", ack.Result)
		}
		return nil, false
	}
	if storageItem, ok, err := network.ParseStorageItemRemoved(pkt); err != nil {
		glog.Errorf("parse storage item removed 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyStorageItemRemoved(ctx, storageItem)
		m.ui.storageWindow.ClampScroll(ctx.Session)
		m.ui.storageWindow.Refresh(ctx, &m.ui.itemInfoWindow)
		return nil, false
	}
	if cartItem, ok, err := network.ParseCartItemRemoved(pkt); err != nil {
		glog.Errorf("parse cart item removed 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyCartItemRemoved(ctx, cartItem)
		m.ui.cartWindow.ClampScroll(ctx.Session)
		m.ui.cartWindow.Refresh(ctx, &m.ui.itemInfoWindow)
		return nil, false
	}
	if vendOpen, ok, err := network.ParseVendingOpenRequest(pkt); err != nil {
		glog.Errorf("parse vending open request 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("vending open request max_items=%d", vendOpen.MaxItems)
		m.ui.vendingWindow.OpenSetup(ctx, vendOpen)
		return nil, false
	}
	if board, ok, err := network.ParseVendingBoard(pkt); err != nil {
		glog.Errorf("parse vending board 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyVendingBoard(ctx, board)
		return nil, false
	}
	if board, ok, err := network.ParseVendingBoardDisappear(pkt); err != nil {
		glog.Errorf("parse vending board disappear 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyVendingBoardDisappear(ctx, board)
		return nil, false
	}
	if vendList, ok, err := network.ParseVendingItemList(pkt); err != nil {
		glog.Errorf("parse vending item list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if vendList.Own {
			m.ui.vendingWindow.ApplyOwnList(ctx, vendList)
		} else {
			m.ui.vendingWindow.OpenBuy(ctx, vendList)
		}
		return nil, false
	}
	if vendResult, ok, err := network.ParseVendingPurchaseResult(pkt); err != nil {
		glog.Errorf("parse vending purchase result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.vendingWindow.ApplyPurchaseResult(ctx, vendResult)
		return nil, false
	}
	if sold, ok, err := network.ParseVendingSoldItem(pkt); err != nil {
		glog.Errorf("parse vending sold item 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.vendingWindow.ApplySoldItem(ctx, sold)
		return nil, false
	}
	if network.ParseStorageClosed(pkt) {
		applyStorageClosed(ctx)
		m.ui.storageWindow.SetOpen(false)
		return nil, false
	}
	if network.ParseCartClosed(pkt) {
		applyCartClosed(ctx)
		m.ui.cartWindow.SetOpen(false)
		return nil, false
	}
	if deal, ok, err := network.ParseShopDealSelection(pkt); err != nil {
		glog.Errorf("parse shop deal selection 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.shopWindow.OpenDeal(deal, ctx)
		return nil, false
	}
	if sellList, ok, err := network.ParseShopSellList(pkt); err != nil {
		glog.Errorf("parse shop sell list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.shopWindow.OpenSell(sellList, ctx)
		return nil, false
	}
	if buyList, ok, err := network.ParseShopBuyList(pkt); err != nil {
		glog.Errorf("parse shop buy list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.shopWindow.OpenBuy(buyList, ctx)
		return nil, false
	}
	if result, ok, err := network.ParseShopResult(pkt); err != nil {
		glog.Errorf("parse shop result 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.shopWindow.ApplyResult(ctx, result)
		if result.Sell && result.Result == 0 {
			m.ui.console.AddBlueMessage("The deal has successfully completed.")
		}
		return nil, false
	}
	if vanish, ok, err := network.ParseActorVanish(pkt); err != nil {
		glog.Errorf("parse actor vanish 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyActorVanish(ctx, vanish)
		if m.pendingAttack.targetID == vanish.ID {
			m.pendingAttack = attackIntent{}
		}
		if m.lockedAttackID == vanish.ID {
			m.clearLockedAttack()
		}
		if m.attackFocusID == vanish.ID {
			m.clearAttackFocus()
		}
		return nil, false
	}
	if resurrection, ok, err := network.ParseActorResurrection(pkt); err != nil {
		glog.Errorf("parse actor resurrection 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyActorResurrection(ctx, resurrection)
		return nil, false
	}
	if look, ok, err := network.ParseActorLookChange(pkt); err != nil {
		glog.Errorf("parse actor look change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		if m.applySkillUnitLookChange(ctx, look) {
			return nil, false
		}
		if applyActorLookChange(ctx, look) {
			m.reloadPlayerSpriteView(ctx, fmt.Sprintf("look type=%d value=%d", look.Type, look.Value))
		}
		return nil, false
	}
	if direction, ok, err := network.ParseActorDirectionChange(pkt); err != nil {
		glog.Errorf("parse actor direction change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyActorDirectionChange(ctx, direction)
		return nil, false
	}
	if state, ok, err := network.ParseActorStateChange(pkt); err != nil {
		glog.Errorf("parse actor state change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyActorStateChange(ctx, state)
		return nil, false
	}
	if bladeStop, ok, err := network.ParseActorBladeStop(pkt); err != nil {
		glog.Errorf("parse actor blade stop 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyActorBladeStop(ctx, bladeStop)
		return nil, false
	}
	if action, ok, err := network.ParseActorActionNotify(pkt); err != nil {
		glog.Errorf("parse actor action 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyActorActionNotify(ctx, action)
		return nil, false
	}
	if life, ok, err := network.ParseActorHPUpdate(pkt); err != nil {
		glog.Errorf("parse actor hp 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyActorHPUpdate(life)
		return nil, false
	}
	if snapshot, ok, err := network.ParseStatusSnapshot(pkt); err != nil {
		glog.Errorf("parse status snapshot 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyStatusSnapshot(ctx, snapshot)
		return nil, false
	}
	if ack, ok, err := network.ParseStatusChangeAck(pkt); err != nil {
		glog.Errorf("parse status change ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.ui.statsWindow.ApplyStatusChangeAck(ctx, ack)
		return nil, false
	}
	if statusEffect, ok, err := network.ParseStatusEffectChange(pkt); err != nil {
		glog.Errorf("parse status effect change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyStatusEffectChange(ctx, statusEffect)
		return nil, false
	}
	if hom, ok, err := network.ParseHomunculusProperty(pkt); err != nil {
		glog.Errorf("parse homunculus property 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyHomunculusProperty(ctx, hom)
		return nil, false
	}
	if feed, ok, err := network.ParseHomunculusFeedResult(pkt); err != nil {
		glog.Errorf("parse homunculus feed 0x%04X: %v", pkt.ID, err)
	} else if ok {
		glog.Debugf("homunculus feed result success=%t item=%d", feed.Result, feed.ItemID)
		m.applyHomunculusFeedResultMessage(ctx, feed)
		return nil, false
	}
	if homState, ok, err := network.ParseHomunculusStateChange(pkt); err != nil {
		glog.Errorf("parse homunculus state 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyHomunculusStateChange(ctx, homState)
		return nil, false
	}
	if homParam, ok, err := network.ParseHomunculusParamChange(pkt); err != nil {
		glog.Errorf("parse homunculus param 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyHomunculusParamChange(ctx, homParam)
		return nil, false
	}
	if homSkills, ok, err := network.ParseHomunculusSkillInfoList(pkt); err != nil {
		glog.Errorf("parse homunculus skill list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyHomunculusSkillInfoList(ctx, homSkills)
		return nil, false
	}
	if homSkill, ok, err := network.ParseHomunculusSkillInfoUpdate(pkt); err != nil {
		glog.Errorf("parse homunculus skill update 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyHomunculusSkillInfoUpdate(ctx, homSkill)
		return nil, false
	}
	if merc, ok, err := network.ParseMercenaryProperty(pkt); err != nil {
		glog.Errorf("parse mercenary property 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyMercenaryProperty(ctx, merc)
		return nil, false
	}
	if mercParam, ok, err := network.ParseMercenaryParamChange(pkt); err != nil {
		glog.Errorf("parse mercenary param 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyMercenaryParamChange(ctx, mercParam)
		return nil, false
	}
	if mercSkills, ok, err := network.ParseMercenarySkillInfoList(pkt); err != nil {
		glog.Errorf("parse mercenary skill list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyMercenarySkillInfoList(ctx, mercSkills)
		return nil, false
	}
	if mercSkill, ok, err := network.ParseMercenarySkillInfoUpdate(pkt); err != nil {
		glog.Errorf("parse mercenary skill update 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applyMercenarySkillInfoUpdate(ctx, mercSkill)
		return nil, false
	}
	if list, ok, err := network.ParseSkillInfoList(pkt); err != nil {
		glog.Errorf("parse skill list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applySkillInfoList(ctx, list)
		m.ui.skillWindow.ClampScroll(ctx.Session)
		return nil, false
	}
	if update, ok, err := network.ParseSkillInfoUpdate(pkt); err != nil {
		glog.Errorf("parse skill update 0x%04X: %v", pkt.ID, err)
	} else if ok {
		applySkillInfoUpdate(ctx, update)
		m.ui.skillWindow.ClampScroll(ctx.Session)
		return nil, false
	}
	if auto, ok, err := network.ParseAutoRunSkill(pkt); err != nil {
		glog.Errorf("parse auto-run skill 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.skills().ApplyAutoRun(ctx, auto)
		return nil, false
	}
	if warpList, ok, err := network.ParseWarpPointList(pkt); err != nil {
		glog.Errorf("parse warp point list 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyWarpPointList(ctx, warpList)
		return nil, false
	}
	if memo, ok, err := network.ParseRememberWarpPointAck(pkt); err != nil {
		glog.Errorf("parse remember warp point ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyRememberWarpPointAck(ctx, memo)
		return nil, false
	}
	if fail, ok, err := network.ParseSkillFailAck(pkt); err != nil {
		glog.Errorf("parse skill fail ack 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySkillFailAck(ctx, fail)
		return nil, false
	}
	if cast, ok, err := network.ParseSkillCastNotify(pkt); err != nil {
		glog.Errorf("parse skill cast 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySkillCastNotify(ctx, cast)
		return nil, false
	}
	if groundSkill, ok, err := network.ParseGroundSkillNotify(pkt); err != nil {
		glog.Errorf("parse ground skill 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyGroundSkillNotify(ctx, groundSkill)
		return nil, false
	}
	if skillUnit, ok, err := network.ParseSkillUnitEntry(pkt); err != nil {
		glog.Errorf("parse skill unit 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySkillUnitEntry(ctx, skillUnit)
		return nil, false
	}
	if skillUnit, ok, err := network.ParseSkillUnitDisappear(pkt); err != nil {
		glog.Errorf("parse skill unit disappear 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySkillUnitDisappear(skillUnit)
		return nil, false
	}
	if skillUnit, ok, err := network.ParseSkillUnitUpdate(pkt); err != nil {
		glog.Errorf("parse skill unit update 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySkillUnitUpdate(skillUnit)
		return nil, false
	}
	if effect, ok, err := network.ParseSpecialEffectNotify(pkt); err != nil {
		glog.Errorf("parse special effect 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySpecialEffectNotify(ctx, effect)
		return nil, false
	}
	if mvp, ok, err := network.ParseMVPNotify(pkt); err != nil {
		glog.Errorf("parse mvp effect 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyMVPNotify(ctx, mvp)
		return nil, false
	}
	if skill, ok, err := network.ParseSkillNoDamageNotify(pkt); err != nil {
		glog.Errorf("parse skill nodamage 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applySkillNoDamageNotify(ctx, skill)
		return nil, false
	}
	if failure, ok, err := network.ParseAttackFailureForDistance(pkt); err != nil {
		glog.Errorf("parse attack distance failure 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyAttackFailureForDistance(ctx, failure)
		return nil, false
	}
	if recovery, ok, err := network.ParseRecovery(pkt); err != nil {
		glog.Errorf("parse recovery 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyRecovery(ctx, recovery)
		return nil, false
	}
	if change, ok, err := network.ParseParameterChange(pkt); err != nil {
		glog.Errorf("parse parameter change 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.applyParameterChange(ctx, change)
		if change.VarID == network.StatusHP {
			m.clearLocalDeathStateIfAlive(ctx)
		}
		return nil, false
	}
	if entry, ok, err := network.ParseActorEntry(pkt); err != nil {
		glog.Errorf("parse actor entry 0x%04X: %v", pkt.ID, err)
	} else if ok {
		m.clearActorDeath(entry.ID)
		m.clearActorVanish(entry.ID)
		m.upsertNetworkActor(ctx, entry)
		m.applyWarpPortalEntry(ctx, entry)
	}

	return nil, false
}
