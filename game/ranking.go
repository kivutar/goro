package game

import (
	"image/color"
	"strconv"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

var fameRankingColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}

type fameRankingPresentation struct {
	profession          string
	professionMessageID int
	pointMessageID      int
	pointFallback       string
}

func (m *WorldMode) applyFameRanking(ctx client.Context, ranking network.FameRanking) {
	presentation := fameRankingPresentationFor(ranking.Kind)
	profession := rankingMessageString(ctx, presentation.professionMessageID, presentation.profession)
	rank := rankingMessageString(ctx, 2383, "Rank")
	points := rankingMessageString(ctx, 2385, "Points")

	m.ui.console.AddColoredMessage(fameRankingColor, "=========== %s %s ===========", profession, rank)
	for index, entry := range ranking.Entries {
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			name = "None"
		}
		m.ui.console.AddColoredMessage(fameRankingColor, "[%d] %s : %d %s", index+1, name, entry.Points, points)
	}
}

func (m *WorldMode) applyFamePointUpdate(ctx client.Context, update network.FamePointUpdate) {
	presentation := fameRankingPresentationFor(update.Kind)
	message := rankingMessageString(ctx, presentation.pointMessageID, presentation.pointFallback)
	message = strings.Replace(message, "%d", strconv.FormatUint(uint64(update.GainedPoints), 10), 1)
	message = strings.Replace(message, "%d", strconv.FormatUint(uint64(update.TotalPoints), 10), 1)
	m.ui.console.AddSystemMessage("%s", message)
	glog.Debugf("fame points kind=%d gained=%d total=%d", update.Kind, update.GainedPoints, update.TotalPoints)
}

func fameRankingPresentationFor(kind network.FameRankingKind) fameRankingPresentation {
	switch kind {
	case network.FameRankingBlacksmith:
		return fameRankingPresentation{
			profession:          "BlackSmith",
			professionMessageID: 2386,
			pointMessageID:      901,
			pointFallback:       "[Point] You have been rewarded with %d Blacksmith rank points. Your point total is %d.",
		}
	case network.FameRankingAlchemist:
		return fameRankingPresentation{
			profession:          "Alchemist",
			professionMessageID: 2387,
			pointMessageID:      902,
			pointFallback:       "[Point] You have been rewarded with %d Alchemist rank points. Your point total is %d.",
		}
	case network.FameRankingTaekwon:
		return fameRankingPresentation{
			profession:          "Taekwon",
			professionMessageID: 2388,
			pointMessageID:      926,
			pointFallback:       "[POINT] You have been rewarded with %d Tae-Kwon Mission rank points. Your point total is %d.",
		}
	default:
		return fameRankingPresentation{
			profession:          "Unknown",
			professionMessageID: -1,
			pointMessageID:      -1,
			pointFallback:       "[Point] You have been rewarded with %d rank points. Your point total is %d.",
		}
	}
}

func rankingMessageString(ctx client.Context, id int, fallback string) string {
	if ctx.Resources == nil {
		return fallback
	}
	message, ok := ctx.Resources.MsgString(id)
	if !ok || strings.TrimSpace(message) == "" {
		return fallback
	}
	return message
}
