package game

import (
	"fmt"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"image"
	"image/color"
	"math"
	"strings"
	"time"

	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

type sceneItemDrawEntry struct {
	item      worldstate.FloorItem
	billboard *spriteBillboard
	screenX   float64
	screenY   float64
	worldX    float64
	worldY    float64
	worldZ    float64
	scale     float64
	depth     float64
}

type itemSpriteKey struct {
	itemID     uint16
	identified bool
}

const (
	itemBillboardWidth       = 96
	itemBillboardHeight      = 96
	itemBillboardAnchorX     = 48
	itemBillboardAnchorY     = 72
	groundItemScreenScale    = 0.8
	itemDropStartOffset      = 2.2
	itemDropStartSpeed       = -0.28
	itemDropGravityPerFrame  = 0.18
	itemDropBounceFactor     = 0.35
	itemDropSpeedDampening   = 0.45
	itemDropStopSpeed        = 0.45
	itemDropSimulationFPS    = 60
	itemDropSimulationFrames = 120
	pickupAnimationDuration  = 450 * time.Millisecond
)

var itemNameLabelForeground = color.RGBA{R: 255, G: 239, B: 148, A: 255}

func (m *WorldMode) applyFloorItemEntry(ctx client.Context, entry network.FloorItemEntry) {
	if ctx.World == nil {
		return
	}
	item := worldstate.FloorItem{
		ID:         entry.ID,
		ItemID:     entry.ItemID,
		Identified: entry.Identified,
		X:          entry.X,
		Y:          entry.Y,
		SubX:       entry.SubX,
		SubY:       entry.SubY,
		Amount:     entry.Amount,
		Falling:    entry.Falling,
		DroppedAt:  time.Now(),
	}
	ctx.World.UpsertItem(item)
	glog.Debugf("floor item entry id=%d item_id=%d identified=%t amount=%d x=%d y=%d sub=%d,%d falling=%t", item.ID, item.ItemID, item.Identified, item.Amount, item.X, item.Y, item.SubX, item.SubY, item.Falling)
}

func (m *WorldMode) applyFloorItemDisappear(ctx client.Context, disappear network.FloorItemDisappear) {
	if ctx.World == nil {
		return
	}
	ctx.World.RemoveItem(disappear.ID)
	if m.pendingPickup.itemID == disappear.ID {
		m.pendingPickup = pickupIntent{}
	}
	glog.Debugf("floor item disappear id=%d", disappear.ID)
}

func (m *WorldMode) applyItemPickupAck(ctx client.Context, ack network.ItemPickupAck) {
	if ack.Result == 0 {
		m.pendingPickup = pickupIntent{}
		addPickedSessionInventoryItem(ctx.Session, session.InventoryItem{
			Index:      ack.Index,
			ItemID:     ack.ItemID,
			Type:       ack.Type,
			Location:   ack.Location,
			Identified: ack.Identified,
			Amount:     maxInt(1, int(ack.Amount)),
			Equip:      inventoryItemTypeIsEquipment(ack.Type),
			Damaged:    ack.Damaged,
			Refine:     ack.Refine,
		})
		m.applyLocalPickupSuccess(ctx)
		glog.Debugf("item pickup ack success index=%d item_id=%d amount=%d type=%d location=0x%04X identified=%t", ack.Index, ack.ItemID, ack.Amount, ack.Type, ack.Location, ack.Identified)
		return
	}
	glog.Warnf("item pickup ack failed index=%d item_id=%d amount=%d result=%d", ack.Index, ack.ItemID, ack.Amount, ack.Result)
}

func (m *WorldMode) requestPickup(ctx client.Context, item worldstate.FloorItem, source string) {
	if ctx.Network == nil {
		m.setWalkCooldown(walkErrorCooldown)
		return
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	if itemWithinPickupRange(playerX, playerY, item.X, item.Y) {
		m.sendPickupRequest(ctx, item, source)
		return
	}
	targetX, targetY, ok := pickupApproachCell(ctx, item)
	if !ok {
		glog.Warnf("%s pickup walk blocked item=%d player=%d,%d item=%d,%d", source, item.ID, playerX, playerY, item.X, item.Y)
		m.setWalkCooldown(walkRequestCooldown)
		return
	}
	m.pendingPickup = pickupIntent{
		itemID:  item.ID,
		expires: time.Now().Add(8 * time.Second),
	}
	glog.Debugf("%s pickup walk target item=%d player=%d,%d item=%d,%d walk=%d,%d", source, item.ID, playerX, playerY, item.X, item.Y, targetX, targetY)
	m.requestWalk(ctx, targetX, targetY, source+" pickup")
}

func (m *WorldMode) continuePendingPickup(ctx client.Context, source string) {
	m.updatePendingPickup(ctx, source, true)
}

func (m *WorldMode) updatePendingPickup(ctx client.Context, source string, logOutOfRange bool) {
	if m.pendingPickup.itemID == 0 || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(m.pendingPickup.expires) {
		glog.Debugf("%s pending pickup expired item=%d", source, m.pendingPickup.itemID)
		m.pendingPickup = pickupIntent{}
		return
	}
	item, ok := ctx.World.Items[m.pendingPickup.itemID]
	if !ok {
		glog.Debugf("%s pending pickup item vanished id=%d", source, m.pendingPickup.itemID)
		m.pendingPickup = pickupIntent{}
		return
	}
	playerX, playerY := currentPlayerCell(ctx, now)
	if !itemWithinPickupRange(playerX, playerY, item.X, item.Y) {
		if logOutOfRange {
			glog.Debugf("%s pending pickup still out of range item=%d player=%d,%d item=%d,%d", source, item.ID, playerX, playerY, item.X, item.Y)
		}
		return
	}
	readyAt := pendingPickupReadyAt(ctx.World.Player, now)
	if m.pendingPickup.readyAt.IsZero() {
		m.pendingPickup.readyAt = readyAt
	}
	if logOutOfRange {
		glog.Debugf("%s pending pickup scheduled item=%d delay_ms=%d", source, item.ID, maxInt(0, int(m.pendingPickup.readyAt.Sub(now).Milliseconds())))
	}
}

func (m *WorldMode) processPendingPickup(ctx client.Context) {
	if m.pendingPickup.itemID == 0 || m.pendingPickup.readyAt.IsZero() || ctx.World == nil {
		return
	}
	now := time.Now()
	if now.After(m.pendingPickup.expires) {
		glog.Debugf("pending pickup expired item=%d", m.pendingPickup.itemID)
		m.pendingPickup = pickupIntent{}
		return
	}
	if now.Before(m.pendingPickup.readyAt) {
		return
	}
	item, ok := ctx.World.Items[m.pendingPickup.itemID]
	if !ok {
		glog.Debugf("pending pickup item vanished id=%d", m.pendingPickup.itemID)
		m.pendingPickup = pickupIntent{}
		return
	}
	playerX, playerY := currentPlayerCell(ctx, now)
	if !itemWithinPickupRange(playerX, playerY, item.X, item.Y) {
		glog.Debugf("pending pickup became out of range item=%d player=%d,%d item=%d,%d", item.ID, playerX, playerY, item.X, item.Y)
		m.pendingPickup.readyAt = time.Time{}
		m.requestPickup(ctx, item, "pending")
		return
	}
	m.pendingPickup = pickupIntent{}
	m.sendPickupRequest(ctx, item, "pending")
}

func pendingPickupReadyAt(player worldstate.Actor, now time.Time) time.Time {
	readyAt := now.Add(60 * time.Millisecond)
	if actorIsMovingAt(player, now) && player.MoveDuration > 0 {
		walkReadyAt := player.MoveStarted.Add(player.MoveDuration).Add(60 * time.Millisecond)
		if walkReadyAt.After(readyAt) {
			readyAt = walkReadyAt
		}
	}
	return readyAt
}

func (m *WorldMode) sendPickupRequest(ctx client.Context, item worldstate.FloorItem, source string) {
	if err := ctx.Network.SendItemPickup(item.ID); err == nil {
		m.pickupReqItemID = item.ID
		m.setWalkCooldown(walkRequestCooldown)
	} else {
		glog.Warnf("%s pickup request failed item=%d: %v", source, item.ID, err)
		m.setWalkCooldown(walkErrorCooldown)
	}
}

func (m *WorldMode) applyLocalPickupSuccess(ctx client.Context) {
	if ctx.World == nil {
		return
	}
	itemID := m.pickupReqItemID
	m.pickupReqItemID = 0
	if itemID != 0 {
		if item, ok := ctx.World.Items[itemID]; ok {
			m.facePlayerTowardItem(ctx, item)
			ctx.World.RemoveItem(itemID)
		}
	}
	m.startLocalPickupAnimation(ctx, time.Now())
}

func (m *WorldMode) startLocalPickupAnimation(ctx client.Context, started time.Time) {
	if ctx.Session == nil {
		return
	}
	if ctx.Session.AccountID != 0 {
		m.startActorAnimation(ctx, ctx.Session.AccountID, spriteActionPickup, started, pickupAnimationDuration)
	}
	if ctx.Session.CharID != 0 {
		m.startActorAnimation(ctx, ctx.Session.CharID, spriteActionPickup, started, pickupAnimationDuration)
	}
}

func (m *WorldMode) facePlayerTowardItem(ctx client.Context, item worldstate.FloorItem) {
	if ctx.World == nil {
		return
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	dir := directionFromDelta(playerX, playerY, item.X, item.Y, ctx.World.Dir)
	ctx.World.Player.Dir = dir
	ctx.World.Dir = dir
	if ctx.Session != nil {
		ctx.Session.PlayerDir = dir
	}
}

func itemWithinPickupRange(playerX, playerY, itemX, itemY int) bool {
	return maxInt(absInt(playerX-itemX), absInt(playerY-itemY)) <= 1
}

func pickupApproachCell(ctx client.Context, item worldstate.FloorItem) (int, int, bool) {
	if walkTargetInBounds(ctx, item.X, item.Y) {
		return item.X, item.Y, true
	}
	playerX, playerY := currentPlayerCell(ctx, time.Now())
	bestX, bestY := 0, 0
	bestDistance := math.Inf(1)
	found := false
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			x, y := item.X+dx, item.Y+dy
			if !walkTargetInBounds(ctx, x, y) {
				continue
			}
			dist := float64((playerX-x)*(playerX-x) + (playerY-y)*(playerY-y))
			if !found || dist < bestDistance {
				found = true
				bestDistance = dist
				bestX = x
				bestY = y
			}
		}
	}
	return bestX, bestY, found
}

func (m *WorldMode) drawGroundItems(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	for _, entry := range m.collectSceneItemEntries(screen, ctx, projection, now) {
		m.drawGroundItemEntry3D(screen, projection, entry)
	}
}

func (m *WorldMode) collectSceneItemEntries(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) []sceneItemDrawEntry {
	if ctx.World == nil || len(ctx.World.Items) == 0 {
		return nil
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	entries := make([]sceneItemDrawEntry, 0, len(ctx.World.Items))
	for _, item := range ctx.World.Items {
		x, y := floorItemWorldPosition(item)
		z := floorItemRenderHeight(ctx.World, item, now)
		point := projection.Project(cellCenter(x), cellCenter(y), z)
		if point.x < -48 || point.y < -80 || point.x > float32(width+48) || point.y > float32(height+48) {
			continue
		}
		scale := actorBillboardScreenScale(projection, cellCenter(x), cellCenter(y), z) * groundItemScreenScale
		entries = append(entries, sceneItemDrawEntry{
			item:      item,
			billboard: m.itemSpriteBillboard(ctx.Resources, item, now),
			screenX:   float64(point.x),
			screenY:   float64(point.y),
			worldX:    cellCenter(x),
			worldY:    cellCenter(y),
			worldZ:    z,
			scale:     scale,
			depth:     projection.Depth(cellCenter(x), cellCenter(y), z),
		})
	}
	return entries
}

func (m *WorldMode) drawGroundItemEntry3D(screen *render.Frame, projection sceneProjection, entry sceneItemDrawEntry) {
	if entry.billboard != nil {
		drawSpriteBillboardAlpha3D(screen, projection, entry.billboard, entry.worldX, entry.worldY, entry.worldZ, entry.scale, 1, 1)
		return
	}
	m.drawFallbackGroundItemMarker(screen, entry)
}

func (m *WorldMode) drawHoveredGroundItemLabel(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	if ctx.Input == nil {
		return
	}
	item, ok := clickedGroundItem(ctx, projection, ctx.Input.MouseX, ctx.Input.MouseY, now)
	if !ok {
		return
	}
	label := m.groundItemLabel(ctx, item)
	x, y := floorItemWorldPosition(item)
	z := floorItemRenderHeight(ctx.World, item, now)
	point := projection.Project(cellCenter(x), cellCenter(y), z)
	scale := actorBillboardScreenScale(projection, cellCenter(x), cellCenter(y), z) * groundItemScreenScale
	drawGroundItemNameLabel(screen, label, float64(point.x), float64(point.y), scale)
}

func clickedGroundItem(ctx client.Context, projection sceneProjection, mouseX, mouseY int, now time.Time) (worldstate.FloorItem, bool) {
	if ctx.World == nil || len(ctx.World.Items) == 0 {
		return worldstate.FloorItem{}, false
	}
	bestDistance := math.Inf(1)
	var best worldstate.FloorItem
	for _, item := range ctx.World.Items {
		x, y := floorItemWorldPosition(item)
		z := floorItemRenderHeight(ctx.World, item, now)
		point := projection.Project(cellCenter(x), cellCenter(y), z)
		scale := actorBillboardScreenScale(projection, cellCenter(x), cellCenter(y), z) * 0.42
		if !pointInGroundItemPickBounds(float64(mouseX), float64(mouseY), float64(point.x), float64(point.y), scale) {
			continue
		}
		dx := float64(point.x) - float64(mouseX)
		dy := float64(point.y) - float64(mouseY)
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			bestDistance = distance
			best = item
		}
	}
	return best, bestDistance < math.Inf(1)
}

func pointInGroundItemPickBounds(mouseX, mouseY, centerX, centerY, scale float64) bool {
	scale = normalizePickScale(scale)
	left := centerX - 18*scale
	right := centerX + 18*scale
	top := centerY - 30*scale
	bottom := centerY + 10*scale
	return mouseX >= left && mouseX <= right && mouseY >= top && mouseY <= bottom
}

func groundItemPickBoundsCenter(centerX, centerY, scale float64) (float64, float64) {
	scale = normalizePickScale(scale)
	top := centerY - 30*scale
	bottom := centerY + 10*scale
	return centerX, (top + bottom) / 2
}

func floorItemWorldPosition(item worldstate.FloorItem) (float64, float64) {
	return float64(item.X) - 0.5 + float64(item.SubX)/12, float64(item.Y) - 0.5 + float64(item.SubY)/12
}

func floorItemRenderHeight(world *worldstate.World, item worldstate.FloorItem, now time.Time) float64 {
	x, y := floorItemWorldPosition(item)
	ground := terrainHeightAt(world, x, y)
	if !item.Falling || item.DroppedAt.IsZero() {
		return ground
	}
	return ground + itemDropBounceOffset(item.DroppedAt, now)
}

func (m *WorldMode) groundItemLabel(ctx client.Context, item worldstate.FloorItem) string {
	name := ""
	if ctx.Resources != nil {
		if resolved, ok := ctx.Resources.ItemDisplayName(int(item.ItemID), item.Identified); ok {
			name = resolved
		}
	}
	return res.FormatGroundItemLabel(name, int(item.Amount))
}

func drawGroundItemNameLabel(screen *render.Frame, label string, centerX, baseY, scale float64) {
	label = strings.TrimSpace(label)
	if label == "" {
		return
	}
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	labelY := baseY + 12*scale
	render.DrawCenteredUIOutlinedTextAt(screen, label, centerX, labelY, itemNameLabelForeground, color.RGBA{A: 196})
}

func (m *WorldMode) itemSpriteBillboard(manager *res.Manager, item worldstate.FloorItem, now time.Time) *spriteBillboard {
	view := m.itemSpriteView(manager, item.ItemID, item.Identified)
	if view == nil || view.act == nil || len(view.act.Actions) == 0 || len(view.act.Actions[0].Animations) == 0 {
		return nil
	}
	action := view.act.Actions[0]
	motion := spriteMotionIndex(action, item.DroppedAt, now, true)
	key := singleSpriteBillboardKey{actionIndex: 0, motion: motion}
	if billboard, ok := view.billboards[key]; ok {
		return billboard
	}
	billboard, ok := composeGroundItemBillboard(view, action.Animations[motion])
	if !ok {
		return nil
	}
	view.billboards[key] = billboard
	return billboard
}

func (m *WorldMode) drawInventoryItemIcon(screen *render.Frame, manager *res.Manager, item session.InventoryItem, x, y int) {
	if screen == nil {
		return
	}
	if icon := m.itemIconTexture(manager, item.ItemID, item.Identified); icon != nil {
		bounds := icon.Bounds()
		width, height := float64(bounds.Dx()), float64(bounds.Dy())
		if width > 0 && height > 0 {
			scale := math.Min(float64(inventoryIconSize)/width, float64(inventoryIconSize)/height)
			dstW, dstH := width*scale, height*scale
			var opts render.DrawImageOptions
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(float64(x)+(float64(inventoryIconSize)-dstW)/2, float64(y)+(float64(inventoryIconSize)-dstH)/2)
			opts.Filter = spriteDrawFilter()
			screen.DrawImage(icon, &opts)
			return
		}
	}
	billboard := m.itemSpriteBillboard(manager, worldstate.FloorItem{
		ItemID:     item.ItemID,
		Identified: item.Identified,
		DroppedAt:  time.Time{},
	}, time.Now())
	if billboard == nil || billboard.image == nil {
		m.drawFallbackInventoryItemIcon(screen, x, y)
		return
	}
	bounds := visibleImageBounds(billboard.image)
	if bounds.Empty() {
		m.drawFallbackInventoryItemIcon(screen, x, y)
		return
	}
	width, height := float64(bounds.Dx()), float64(bounds.Dy())
	if width <= 0 || height <= 0 {
		return
	}
	scale := math.Min(float64(inventoryIconSize-2)/width, float64(inventoryIconSize-2)/height)
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	dstW, dstH := width*scale, height*scale
	dstX := float64(x) + (float64(inventoryIconSize)-dstW)/2
	dstY := float64(y) + (float64(inventoryIconSize)-dstH)/2
	vertices := []render.Vertex{
		{DstX: float32(dstX), DstY: float32(dstY), SrcX: float32(bounds.Min.X), SrcY: float32(bounds.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX + dstW), DstY: float32(dstY), SrcX: float32(bounds.Max.X), SrcY: float32(bounds.Min.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX), DstY: float32(dstY + dstH), SrcX: float32(bounds.Min.X), SrcY: float32(bounds.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(dstX + dstW), DstY: float32(dstY + dstH), SrcX: float32(bounds.Max.X), SrcY: float32(bounds.Max.Y), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
	}
	screen.DrawTrianglesOwned(vertices, quadIndices012213, billboard.image, &render.DrawTrianglesOptions{Filter: spriteDrawFilter(), Address: render.AddressClampToZero})
}

func (m *WorldMode) itemIconTexture(manager *res.Manager, itemID uint16, identified bool) *render.Image {
	if manager == nil || itemID == 0 {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(itemID), identified)
	if !ok {
		return nil
	}
	key := fmt.Sprintf("__item_icon_%t_%s", identified, resourceName)
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemIconTextureCandidates(resourceName))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.textures[key] = texture
	return texture
}

func (m *WorldMode) itemCollectionTexture(manager *res.Manager, itemID uint16, identified bool) *render.Image {
	if manager == nil || itemID == 0 {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(itemID), identified)
	if !ok {
		return nil
	}
	key := fmt.Sprintf("__item_collection_%t_%s", identified, resourceName)
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.ItemCollectionTextureCandidates(resourceName))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.textures[key] = texture
	return texture
}

func (m *WorldMode) drawSkillIcon(screen *render.Frame, manager *res.Manager, skill session.Skill, x, y, size int) {
	if screen == nil || size <= 0 {
		return
	}
	if icon := m.skillIconTexture(manager, skill); icon != nil {
		bounds := icon.Bounds()
		width, height := float64(bounds.Dx()), float64(bounds.Dy())
		if width > 0 && height > 0 {
			scale := math.Min(float64(size)/width, float64(size)/height)
			dstW, dstH := width*scale, height*scale
			var opts render.DrawImageOptions
			opts.GeoM.Scale(scale, scale)
			opts.GeoM.Translate(float64(x)+(float64(size)-dstW)/2, float64(y)+(float64(size)-dstH)/2)
			opts.Filter = spriteDrawFilter()
			screen.DrawImage(icon, &opts)
			return
		}
	}
}

func (m *WorldMode) skillIconTexture(manager *res.Manager, skill session.Skill) *render.Image {
	if manager == nil || skill.ID == 0 {
		return nil
	}
	resourceName, ok := manager.SkillResourceName(int(skill.ID))
	if !ok {
		resourceName = strings.ToUpper(strings.ReplaceAll(skillLabel(skill), " ", "_"))
	}
	key := fmt.Sprintf("__skill_icon_%d_%s", skill.ID, resourceName)
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.SkillIconTextureCandidates(resourceName, int(skill.ID)))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(img)
	m.textures[key] = texture
	return texture
}

func (m *WorldMode) drawFallbackInventoryItemIcon(screen *render.Frame, x, y int) {
	img := m.itemMarkerTexture()
	if img == nil {
		return
	}
	bounds := img.Bounds()
	scale := float64(inventoryIconSize) / float64(maxInt(bounds.Dx(), bounds.Dy()))
	var opts render.DrawImageOptions
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(x), float64(y))
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(img, &opts)
}

func visibleImageBounds(img *render.Image) image.Rectangle {
	rgba := img.RGBA()
	if rgba == nil {
		return image.Rectangle{}
	}
	bounds := rgba.Bounds()
	visible := image.Rectangle{Min: bounds.Max, Max: bounds.Min}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if rgba.RGBAAt(x, y).A == 0 {
				continue
			}
			if x < visible.Min.X {
				visible.Min.X = x
			}
			if y < visible.Min.Y {
				visible.Min.Y = y
			}
			if x+1 > visible.Max.X {
				visible.Max.X = x + 1
			}
			if y+1 > visible.Max.Y {
				visible.Max.Y = y + 1
			}
		}
	}
	if visible.Min.X > visible.Max.X || visible.Min.Y > visible.Max.Y {
		return image.Rectangle{}
	}
	return visible.Inset(-1).Intersect(bounds)
}

func (m *WorldMode) itemSpriteView(manager *res.Manager, itemID uint16, identified bool) *spriteView {
	if manager == nil || itemID == 0 {
		return nil
	}
	if m.itemViews == nil {
		m.itemViews = make(map[itemSpriteKey]*spriteView)
	}
	if m.itemViewMiss == nil {
		m.itemViewMiss = make(map[itemSpriteKey]struct{})
	}
	key := itemSpriteKey{itemID: itemID, identified: identified}
	if view := m.itemViews[key]; view != nil {
		return view
	}
	if _, ok := m.itemViewMiss[key]; ok {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(itemID), identified)
	if !ok {
		m.itemViewMiss[key] = struct{}{}
		glog.Warnf("item sprite resource missing item_id=%d identified=%t", itemID, identified)
		return nil
	}
	view, status := loadSpriteView(
		manager,
		res.ItemSpriteResourceCandidates(resourceName, "act"),
		res.ItemSpriteResourceCandidates(resourceName, "spr"),
		nil,
		fmt.Sprintf("item %d", itemID),
	)
	if view == nil {
		m.itemViewMiss[key] = struct{}{}
		glog.Warnf("item sprite unavailable item_id=%d identified=%t resource=%q: %s", itemID, identified, resourceName, status)
		return nil
	}
	m.itemViews[key] = view
	glog.Debugf("item sprite resources item_id=%d identified=%t resource=%q %s", itemID, identified, resourceName, status)
	return view
}

func composeGroundItemBillboard(view *spriteView, anim res.ACTAnimation) (*spriteBillboard, bool) {
	target := render.NewImage(itemBillboardWidth, itemBillboardHeight)
	if !drawSpriteAnimation(target, view, anim, itemBillboardAnchorX, itemBillboardAnchorY, 0, 0) {
		return nil, false
	}
	return &spriteBillboard{
		image:   target,
		anchorX: itemBillboardAnchorX,
		anchorY: itemBillboardAnchorY,
	}, true
}

func (m *WorldMode) drawFallbackGroundItemMarker(screen *render.Frame, entry sceneItemDrawEntry) {
	img := m.itemMarkerTexture()
	if img == nil {
		return
	}
	scale := entry.scale / groundItemScreenScale * 0.42
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		scale = 1
	}
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(-width)/2, float64(-height)+4)
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(entry.screenX, entry.screenY)
	opts.Filter = spriteDrawFilter()
	screen.DrawImage(img, &opts)
}

func itemDropBounceOffset(started, now time.Time) float64 {
	elapsed := now.Sub(started)
	if elapsed <= 0 {
		return itemDropStartOffset
	}
	frames := int(elapsed.Seconds() * itemDropSimulationFPS)
	if frames > itemDropSimulationFrames {
		frames = itemDropSimulationFrames
	}
	pos := itemDropStartOffset
	speed := itemDropStartSpeed
	for i := 0; i < frames; i++ {
		speed += itemDropGravityPerFrame
		pos -= speed
		if pos <= 0 {
			if speed > itemDropStopSpeed {
				pos = -pos * itemDropBounceFactor
				speed = -speed * itemDropSpeedDampening
			} else {
				return 0
			}
		}
	}
	if pos < 0 {
		return 0
	}
	return pos
}

func (m *WorldMode) itemMarkerTexture() *render.Image {
	if m.itemMarker != nil {
		return m.itemMarker
	}
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - 16
			dy := float64(y) - 14
			r := math.Sqrt(dx*dx/1.8 + dy*dy)
			if r > 12 {
				continue
			}
			alpha := uint8(210)
			if r > 9 {
				alpha = uint8(180 - (r-9)*35)
			}
			img.SetRGBA(x, y, color.RGBA{R: 245, G: 220, B: 96, A: alpha})
		}
	}
	for y := 8; y < 22; y++ {
		for x := 10; x < 23; x++ {
			if x == 10 || x == 22 || y == 8 || y == 21 {
				img.SetRGBA(x, y, color.RGBA{R: 80, G: 48, B: 16, A: 230})
			}
		}
	}
	m.itemMarker = render.NewImageFromImage(img)
	return m.itemMarker
}
