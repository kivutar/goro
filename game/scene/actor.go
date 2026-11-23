package scene

import (
	"math"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func ActorBillboardScreenScale(projection Projection, x, y, z float64) float64 {
	_, _, worldUnitsPerScreenPixel, ok := projection.BillboardBasis(x, y, z)
	if !ok {
		return 1
	}
	scale := ActorBillboardWorldUnitsPerSourcePixel / worldUnitsPerScreenPixel
	if scale <= 0 || math.IsNaN(scale) || math.IsInf(scale, 0) {
		return 1
	}
	return scale
}

func ActorRepresentsPlayer(actor worldstate.Actor) bool {
	if actor.HasObjectType {
		return actor.ObjectType == ActorObjectTypePC
	}
	// The 2008 PC-only 0x02ED/0x02EE entries omit the object-type byte.
	// Require their full appearance payload so a move-only update with the
	// zero-value Novice job cannot be mistaken for a player.
	return actor.Appearance && res.HasPlayerJobToken(int(actor.Job))
}

func IsLocalActor(ctx client.Context, id uint32) bool {
	return ctx.Session != nil && id != 0 && (id == ctx.Session.AccountID || id == ctx.Session.CharID)
}

type ActorDrawEntry struct {
	Actor       worldstate.Actor
	ScreenX     float64
	ScreenY     float64
	WorldX      float64
	WorldY      float64
	WorldZ      float64
	Scale       float64
	Shadow      LightTint
	CastShadow  bool
	ShadowScale float64
	ShadowDepth float64
	Depth       float64
	IsPlayer    bool
	Stealth     StealthView
}

const (
	// RO clients map 175 source pixels to the five world units of one map cell.
	actorBillboardSourcePixelsPerCell      = 175.0
	ActorBillboardWorldUnitsPerSourcePixel = actorBillboardCellWorldUnits / actorBillboardSourcePixelsPerCell
)

const (
	actorBillboardCellWorldUnits  = 5.0
	ActorBillboardWorldHeightUnit = 1.0 * actorBillboardCellWorldUnits
	BabyPlayerBodyScale           = 0.8
	ActorJobWarpPortal            = 45
	ActorJobWarpPortalActive      = 128
	ActorJobWarpPortalWaiting     = 129
	ActorJobHiddenNPC             = 111
	ActorJobHiddenWarpNPC         = 139
	ActorJobClearNPC              = 844
	ActorObjectTypePC             = 0
	ActorObjectTypeDisguised      = 1
	ActorObjectTypeMob            = 5
	ActorObjectTypeNPC            = 6
	ActorObjectTypePet            = 7
	ActorObjectTypeHomunculus     = 8
	ActorObjectTypeMercenary      = 9
	ActorObjectTypeElemental      = 10
	ActorObjectTypeUnit           = 11
	ActorObjectTypeNPC2           = 12
	ActorObjectTypeNPCABR         = 13
	ActorObjectTypeNPCBionic      = 14
)

func IsMonsterLikeHoverActor(actor worldstate.Actor) bool {
	if res.HasPlayerJobToken(int(actor.Job)) {
		return false
	}
	job := int(actor.Job)
	return job >= 1000 && (job < 6001 || job > 6047)
}

func SanitizeActorName(name string) string {
	name = strings.TrimSpace(name)
	if hash := strings.IndexByte(name, '#'); hash >= 0 {
		name = strings.TrimSpace(name[:hash])
	}
	if strings.EqualFold(name, "actor") {
		return ""
	}
	return name
}
