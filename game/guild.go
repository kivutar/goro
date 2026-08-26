package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
)

const guildPermissionInvite uint32 = 0x01

func guildCanInvitePlayer(s *session.Session, targetGuildID uint32) bool {
	if s == nil || targetGuildID != 0 {
		return false
	}
	guildID := s.GuildID
	if guildID == 0 {
		guildID = s.Guild.ID
	}
	return guildID != 0 && s.Guild.Right&guildPermissionInvite != 0
}

func guildCanManageRelations(s *session.Session, targetGuildID uint32) bool {
	if s == nil || !s.Guild.IsMaster || targetGuildID == 0 {
		return false
	}
	guildID := s.Guild.ID
	if guildID == 0 {
		guildID = s.GuildID
	}
	return guildID != 0 && guildID != targetGuildID
}

func (m *WorldMode) sendGuildAllianceRequest(ctx client.Context, actorID uint32, name string) {
	if actorID == 0 || ctx.Session == nil || ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild alliance request failed: not connected.")
		return
	}
	if err := ctx.Network.SendGuildAllianceRequest(actorID, ctx.Session.AccountID, ctx.Session.CharID); err != nil {
		glog.Warnf("guild alliance request failed target=%d name=%q: %v", actorID, name, err)
		m.ui.console.AddErrorMessage("Guild alliance request failed.")
		return
	}
	m.ui.console.AddSystemMessage("Alliance request sent to %s's guild.", guildDisplayName(name))
}

func (m *WorldMode) sendGuildHostilityRequest(ctx client.Context, actorID uint32, name string) {
	if actorID == 0 || ctx.Network == nil {
		m.ui.console.AddErrorMessage("Declare hostility failed: not connected.")
		return
	}
	if err := ctx.Network.SendGuildHostilityRequest(actorID); err != nil {
		glog.Warnf("declare guild hostility failed target=%d name=%q: %v", actorID, name, err)
		m.ui.console.AddErrorMessage("Declare hostility failed.")
		return
	}
	m.ui.console.AddSystemMessage("Hostility request sent to %s's guild.", guildDisplayName(name))
}

func (m *WorldMode) openGuildAllianceRequest(ctx client.Context, request network.GuildAllianceRequest) {
	name := guildDisplayName(request.GuildName)
	m.ui.guildAllianceRequest.Open(ctx, "Guild Alliance", fmt.Sprintf("Accept an alliance with %s?", name), func() {
		if ctx.Network == nil {
			m.ui.console.AddErrorMessage("Guild alliance reply failed: not connected.")
			return
		}
		if err := ctx.Network.SendGuildAllianceReply(request.AccountID, true); err != nil {
			glog.Warnf("guild alliance accept failed account=%d guild=%q: %v", request.AccountID, request.GuildName, err)
		}
	}, func() {
		if ctx.Network == nil {
			return
		}
		if err := ctx.Network.SendGuildAllianceReply(request.AccountID, false); err != nil {
			glog.Warnf("guild alliance reject failed account=%d guild=%q: %v", request.AccountID, request.GuildName, err)
		}
	})
}

func (m *WorldMode) handleGuildAllianceResult(result network.GuildAllianceResult) {
	switch result.Result {
	case 0:
		m.ui.console.AddErrorMessage("The guilds are already allied.")
	case 1:
		m.ui.console.AddErrorMessage("The alliance request was rejected.")
	case 2:
		m.ui.console.AddBlueMessage("The guild alliance was established.")
	case 3:
		m.ui.console.AddErrorMessage("The other guild cannot accept more alliances.")
	case 4:
		m.ui.console.AddErrorMessage("Your guild cannot accept more alliances.")
	case 5:
		m.ui.console.AddErrorMessage("Guild alliances are disabled.")
	default:
		m.ui.console.AddErrorMessage("Guild alliance request failed.")
	}
}

func (m *WorldMode) handleGuildHostilityResult(result network.GuildHostilityResult) {
	switch result.Result {
	case 0:
		m.ui.console.AddBlueMessage("The guild has been declared hostile.")
	case 1:
		m.ui.console.AddErrorMessage("Your guild cannot declare more hostilities.")
	case 2:
		m.ui.console.AddErrorMessage("That guild is already hostile.")
	case 3:
		m.ui.console.AddErrorMessage("Guild hostility is disabled.")
	default:
		m.ui.console.AddErrorMessage("Declare hostility failed.")
	}
}

func (m *WorldMode) openDeleteGuildRelationConfirm(ctx client.Context, relation session.GuildRelation) {
	if ctx.Session == nil || !ctx.Session.Guild.IsMaster || relation.GuildID == 0 {
		return
	}
	kind := "alliance"
	if relation.Relation == session.GuildRelationOpposition {
		kind = "hostility"
	}
	name := guildDisplayName(relation.Name)
	m.ui.guildRelationConfirm.Open(ctx, "Guild Relations", fmt.Sprintf("End the %s with %s?", kind, name), func() {
		if ctx.Network == nil {
			m.ui.console.AddErrorMessage("Guild relation update failed: not connected.")
			return
		}
		if err := ctx.Network.SendDeleteGuildRelation(relation.GuildID, relation.Relation); err != nil {
			glog.Warnf("delete guild relation failed guild=%d relation=%d name=%q: %v", relation.GuildID, relation.Relation, relation.Name, err)
			m.ui.console.AddErrorMessage("Guild relation update failed.")
		}
	}, nil)
}

func (m *WorldMode) sendGuildInvite(ctx client.Context, actorID uint32, name string) {
	name = strings.TrimSpace(name)
	if actorID == 0 {
		return
	}
	if ctx.Network == nil {
		glog.Warnf("guild invite failed target=%d name=%q: not connected", actorID, name)
		m.ui.console.AddErrorMessage("Guild invitation failed: not connected.")
		return
	}
	accountID, charID := uint32(0), uint32(0)
	if ctx.Session != nil {
		accountID = ctx.Session.AccountID
		charID = ctx.Session.CharID
	}
	if err := ctx.Network.SendGuildInvite(actorID, accountID, charID); err != nil {
		glog.Warnf("guild invite failed target=%d name=%q: %v", actorID, name, err)
		m.ui.console.AddErrorMessage("Guild invitation failed.")
		return
	}
	m.ui.console.AddBlueMessage("%s has received an invitation to join your guild.", guildDisplayName(name))
}

func (m *WorldMode) openGuildInviteRequest(ctx client.Context, request network.GuildInviteRequest) {
	name := guildDisplayName(request.GuildName)
	rawName := strings.TrimSpace(request.GuildName)
	m.ui.guildRequest.Open(ctx, "Guild Invitation", fmt.Sprintf("Would you like to join %s?", name), func() {
		if ctx.Network == nil {
			glog.Warnf("guild invite accept failed: not connected")
			return
		}
		if err := ctx.Network.SendGuildInviteReply(request.GuildID, true); err != nil {
			glog.Warnf("guild invite accept failed guild=%d name=%q: %v", request.GuildID, request.GuildName, err)
			return
		}
		applyLocalGuildName(ctx, rawName)
	}, func() {
		if ctx.Network == nil {
			glog.Warnf("guild invite reject failed: not connected")
			return
		}
		if err := ctx.Network.SendGuildInviteReply(request.GuildID, false); err != nil {
			glog.Warnf("guild invite reject failed guild=%d name=%q: %v", request.GuildID, request.GuildName, err)
		}
	})
}

func (m *WorldMode) handleGuildCreationResult(ctx client.Context, result network.GuildCreationResult) {
	switch result.Result {
	case 0:
		if name := pendingGuildName(ctx); name != "" {
			applyLocalGuildName(ctx, name)
		}
		m.ui.console.AddBlueMessage("Guild created.")
	case 1:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("You are already in a guild.")
	case 2:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("Guild name already exists.")
	case 3:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("You need the required item to create a guild.")
	default:
		clearPendingGuildName(ctx)
		m.ui.console.AddErrorMessage("Guild creation failed.")
	}
}

func (m *WorldMode) handleGuildInviteAck(ack network.GuildInviteAck) {
	switch ack.Result {
	case 0:
		m.ui.console.AddErrorMessage("That character is already in a guild.")
	case 1:
		m.ui.console.AddErrorMessage("Guild invitation was rejected.")
	case 2:
		m.ui.console.AddBlueMessage("Guild invitation accepted.")
	case 3:
		m.ui.console.AddErrorMessage("The guild is full.")
	default:
		m.ui.console.AddErrorMessage("Guild invitation failed.")
	}
}

func (m *WorldMode) setGuildEmblemOptions(ctx client.Context) {
	m.ui.guildWindow.SetEmblemOptions(ctx, localGuildEmblemOptions(ctx.Config.DataDir))
}

func (m *WorldMode) uploadGuildEmblem(ctx client.Context, path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	if ctx.Session == nil || !ctx.Session.Guild.IsMaster {
		m.ui.console.AddErrorMessage("Guild emblem upload failed: guild master only.")
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild emblem upload failed: not connected.")
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		m.ui.console.AddErrorMessage("Guild emblem upload failed: %s", err)
		return
	}
	if len(data) < 2 || data[0] != 'B' || data[1] != 'M' {
		m.ui.console.AddErrorMessage("Guild emblem upload failed: expected BMP file.")
		return
	}
	if len(data) > 1783 {
		m.ui.console.AddErrorMessage("Guild emblem upload failed: BMP is too large.")
		return
	}
	if err := ctx.Network.SendGuildEmblem(data); err != nil {
		m.ui.console.AddErrorMessage("Guild emblem upload failed.")
		glog.Warnf("guild emblem upload failed path=%q: %v", path, err)
		return
	}
	m.ui.console.AddSystemMessage("Guild emblem uploaded.")
}

func (m *WorldMode) changeGuildMemberPositions(ctx client.Context, updates []gameui.GuildMemberPositionUpdate) {
	if len(updates) == 0 {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild member position update failed: not connected.")
		return
	}
	if ctx.Session == nil {
		m.ui.console.AddErrorMessage("Guild member position update failed.")
		return
	}
	existing := make(map[[2]uint32]struct{}, len(ctx.Session.Guild.Members))
	for _, member := range ctx.Session.Guild.Members {
		existing[[2]uint32{member.AccountID, member.CharID}] = struct{}{}
	}
	members := make([]network.GuildMemberPosition, 0, len(updates))
	for _, update := range updates {
		if _, ok := existing[[2]uint32{update.AccountID, update.CharID}]; !ok {
			m.ui.console.AddErrorMessage("Guild member position update failed: member not found.")
			return
		}
		members = append(members, network.GuildMemberPosition{
			AccountID:  update.AccountID,
			CharID:     update.CharID,
			PositionID: update.PositionID,
		})
	}
	if err := ctx.Network.SendGuildMemberPositions(members); err != nil {
		m.ui.console.AddErrorMessage("Guild member position update failed.")
		glog.Warnf("guild member position update failed members=%d: %v", len(members), err)
		return
	}
}

func (m *WorldMode) updateGuildPositions(ctx client.Context, updates []gameui.GuildPositionUpdate) {
	if len(updates) == 0 {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild position update failed: not connected.")
		return
	}
	positions := make([]network.GuildPosition, 0, len(updates))
	for _, update := range updates {
		positions = append(positions, network.GuildPosition{
			PositionID: update.PositionID,
			Right:      update.Right,
			Ranking:    update.Ranking,
			PayRate:    update.PayRate,
			PosName:    update.PosName,
		})
	}
	if err := ctx.Network.SendGuildPositions(positions); err != nil {
		m.ui.console.AddErrorMessage("Guild position update failed.")
		glog.Warnf("guild position update failed positions=%d: %v", len(positions), err)
		return
	}
	if ctx.Session != nil {
		applyLocalGuildPositions(ctx, positions)
		applyLocalGuildPositionNames(ctx, positions)
	}
	m.ui.guildWindow.Refresh(ctx)
}

func (m *WorldMode) updateGuildNotice(ctx client.Context, subject, notice string) {
	subject = strings.TrimSpace(subject)
	notice = strings.TrimSpace(notice)
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild notice update failed: not connected.")
		return
	}
	if ctx.Session == nil {
		m.ui.console.AddErrorMessage("Guild notice update failed.")
		return
	}
	guildID := ctx.Session.Guild.ID
	if guildID == 0 {
		guildID = ctx.Session.GuildID
	}
	if guildID == 0 {
		m.ui.console.AddErrorMessage("Guild notice update failed.")
		return
	}
	if err := ctx.Network.SendGuildNotice(guildID, subject, notice); err != nil {
		m.ui.console.AddErrorMessage("Guild notice update failed.")
		glog.Warnf("guild notice update failed guild=%d subject=%q: %v", guildID, subject, err)
		return
	}
	applyLocalGuildNotice(ctx, network.GuildNotice{Subject: subject, Notice: notice})
	m.ui.guildWindow.Refresh(ctx)
}

func (m *WorldMode) handleGuildNotice(ctx client.Context, notice network.GuildNotice) {
	applyLocalGuildNotice(ctx, notice)
	m.addGuildNoticeMessage(notice.Subject)
	m.addGuildNoticeMessage(notice.Notice)
}

func (m *WorldMode) addGuildNoticeMessage(text string) {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m.ui.console.AddGuildMessage("[ %s ]", line)
	}
}

func (m *WorldMode) levelUpGuildSkills(ctx client.Context, skillIDs []uint16) {
	if len(skillIDs) == 0 {
		return
	}
	if ctx.Network == nil {
		m.ui.console.AddErrorMessage("Guild skill level up failed: not connected.")
		return
	}
	for _, skillID := range skillIDs {
		if skillID == 0 {
			continue
		}
		if err := ctx.Network.SendSkillLevelUp(skillID); err != nil {
			m.ui.console.AddErrorMessage("Guild skill level up failed.")
			glog.Warnf("guild skill level up failed skill=%d: %v", skillID, err)
			return
		}
	}
}

func localGuildEmblemOptions(dataDir string) []gameui.GuildEmblemOption {
	if strings.TrimSpace(dataDir) == "" {
		return nil
	}
	dirs := []string{
		filepath.Join(dataDir, "Emblem"),
		filepath.Join(dataDir, "emblem"),
	}
	seen := make(map[string]struct{})
	var options []gameui.GuildEmblemOption
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".bmp") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			options = append(options, gameui.GuildEmblemOption{
				Label: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
				Path:  path,
			})
		}
	}
	sort.Slice(options, func(i, j int) bool {
		return strings.ToLower(options[i].Label) < strings.ToLower(options[j].Label)
	})
	return options
}

func guildDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Guild"
	}
	return name
}

func pendingGuildName(ctx client.Context) string {
	if ctx.Session == nil {
		return ""
	}
	return strings.TrimSpace(ctx.Session.PendingGuildName)
}

func clearPendingGuildName(ctx client.Context) {
	if ctx.Session != nil {
		ctx.Session.PendingGuildName = ""
	}
}

func applyLocalGuildName(ctx client.Context, name string) {
	applyLocalGuildInfo(ctx, 0, 0, name)
}

func applyLocalGuildBelonging(ctx client.Context, belonging network.GuildBelonging) {
	applyLocalGuildInfo(ctx, belonging.GuildID, belonging.EmblemVersion, belonging.GuildName)
	if ctx.Session != nil {
		ctx.Session.Guild.IsMaster = belonging.IsMaster
		ctx.Session.Guild.Right = belonging.Mode
	}
}

func applyLocalGuildDetails(ctx client.Context, info network.GuildInfo) {
	applyLocalGuildInfo(ctx, info.GuildID, info.EmblemVersion, info.GuildName)
	if ctx.Session != nil {
		isMaster := ctx.Session.Guild.IsMaster
		right := ctx.Session.Guild.Right
		masterName := strings.TrimSpace(info.MasterName)
		if selectedName := strings.TrimSpace(ctx.Session.Selected.Name); selectedName != "" && masterName != "" {
			isMaster = selectedName == masterName
		}
		members := ctx.Session.Guild.Members
		positions := ctx.Session.Guild.Positions
		skillPoints := ctx.Session.Guild.SkillPoints
		skills := ctx.Session.Guild.Skills
		expelHistory := ctx.Session.Guild.ExpelHistory
		noticeSubject := ctx.Session.Guild.NoticeSubject
		notice := ctx.Session.Guild.Notice
		relations := ctx.Session.Guild.Relations
		ctx.Session.Guild = session.Guild{
			ID:               info.GuildID,
			IsMaster:         isMaster,
			Right:            right,
			Level:            info.Level,
			UserNum:          info.UserNum,
			MaxUserNum:       info.MaxUserNum,
			UserAverageLevel: info.UserAverageLevel,
			Exp:              info.Exp,
			MaxExp:           info.MaxExp,
			Point:            info.Point,
			Honor:            info.Honor,
			Virtue:           info.Virtue,
			EmblemVersion:    info.EmblemVersion,
			Name:             strings.TrimSpace(info.GuildName),
			MasterName:       masterName,
			ManageLand:       strings.TrimSpace(info.ManageLand),
			Zeny:             info.Zeny,
			Members:          members,
			Positions:        positions,
			SkillPoints:      skillPoints,
			Skills:           skills,
			ExpelHistory:     expelHistory,
			NoticeSubject:    noticeSubject,
			Notice:           notice,
			Relations:        relations,
		}
	}
}

func applyLocalGuildMembers(ctx client.Context, members []network.GuildMember) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Guild.Members = make([]session.GuildMember, 0, len(members))
	var online uint32
	for _, member := range members {
		if member.CurrentState != 0 {
			online++
		}
		ctx.Session.Guild.Members = append(ctx.Session.Guild.Members, sessionGuildMemberFromNetwork(member))
	}
	ctx.Session.Guild.UserNum = online
	glog.Debugf("guild member list received members=%d online=%d", len(members), online)
}

func applyLocalGuildMember(ctx client.Context, member network.GuildMember) {
	if ctx.Session == nil {
		return
	}
	sessionMember := sessionGuildMemberFromNetwork(member)
	for i := range ctx.Session.Guild.Members {
		if ctx.Session.Guild.Members[i].AccountID == sessionMember.AccountID && ctx.Session.Guild.Members[i].CharID == sessionMember.CharID {
			ctx.Session.Guild.Members[i] = sessionMember
			recountGuildOnlineMembers(&ctx.Session.Guild)
			glog.Debugf("guild member updated account=%d char=%d position=%d", sessionMember.AccountID, sessionMember.CharID, sessionMember.PositionID)
			return
		}
	}
	ctx.Session.Guild.Members = append(ctx.Session.Guild.Members, sessionMember)
	recountGuildOnlineMembers(&ctx.Session.Guild)
	glog.Debugf("guild member added account=%d char=%d position=%d", sessionMember.AccountID, sessionMember.CharID, sessionMember.PositionID)
}

func applyLocalGuildRelations(ctx client.Context, relations []network.GuildRelation) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Guild.Relations = make([]session.GuildRelation, 0, len(relations))
	for _, relation := range relations {
		ctx.Session.Guild.Relations = append(ctx.Session.Guild.Relations, session.GuildRelation{
			Relation: relation.Relation,
			GuildID:  relation.GuildID,
			Name:     strings.TrimSpace(relation.Name),
		})
	}
}

func applyLocalGuildRelation(ctx client.Context, relation network.GuildRelation) {
	if ctx.Session == nil || relation.GuildID == 0 {
		return
	}
	for i := range ctx.Session.Guild.Relations {
		current := &ctx.Session.Guild.Relations[i]
		if current.GuildID == relation.GuildID && current.Relation == relation.Relation {
			current.Name = strings.TrimSpace(relation.Name)
			return
		}
	}
	ctx.Session.Guild.Relations = append(ctx.Session.Guild.Relations, session.GuildRelation{
		Relation: relation.Relation,
		GuildID:  relation.GuildID,
		Name:     strings.TrimSpace(relation.Name),
	})
}

func applyLocalGuildRelationDeleted(ctx client.Context, deleted network.GuildRelationDeleted) {
	if ctx.Session == nil {
		return
	}
	relations := ctx.Session.Guild.Relations
	for i := range relations {
		if relations[i].GuildID != deleted.GuildID || relations[i].Relation != deleted.Relation {
			continue
		}
		copy(relations[i:], relations[i+1:])
		ctx.Session.Guild.Relations = relations[:len(relations)-1]
		return
	}
}

func applyLocalGuildMemberState(ctx client.Context, state network.GuildMemberState) bool {
	if ctx.Session == nil {
		return false
	}
	for i := range ctx.Session.Guild.Members {
		member := &ctx.Session.Guild.Members[i]
		if member.AccountID != state.AccountID || member.CharID != state.CharID {
			continue
		}
		changed := member.CurrentState != state.State
		member.CurrentState = state.State
		if state.HasAppearance {
			changed = changed || member.Sex != state.Sex || member.HeadType != state.HeadType || member.HeadPalette != state.HeadPalette
			member.Sex = state.Sex
			member.HeadType = state.HeadType
			member.HeadPalette = state.HeadPalette
		}
		if !changed {
			return false
		}
		recountGuildOnlineMembers(&ctx.Session.Guild)
		return true
	}
	return false
}

func recountGuildOnlineMembers(guild *session.Guild) {
	if guild == nil {
		return
	}
	var online uint32
	for _, member := range guild.Members {
		if member.Online() {
			online++
		}
	}
	guild.UserNum = online
}

func applyLocalGuildMemberPositions(ctx client.Context, positions []network.GuildMemberPosition) {
	if ctx.Session == nil {
		return
	}
	for _, position := range positions {
		for i := range ctx.Session.Guild.Members {
			member := &ctx.Session.Guild.Members[i]
			if member.AccountID == position.AccountID && member.CharID == position.CharID {
				member.PositionID = position.PositionID
				glog.Debugf("guild member position accepted account=%d char=%d position=%d", position.AccountID, position.CharID, position.PositionID)
				break
			}
		}
	}
}

func sessionGuildMemberFromNetwork(member network.GuildMember) session.GuildMember {
	return session.GuildMember{
		AccountID:    member.AccountID,
		CharID:       member.CharID,
		HeadType:     member.HeadType,
		HeadPalette:  member.HeadPalette,
		Sex:          member.Sex,
		Job:          member.Job,
		Level:        member.Level,
		MemberExp:    member.MemberExp,
		CurrentState: member.CurrentState,
		PositionID:   member.PositionID,
		Memo:         strings.TrimSpace(member.Memo),
		CharName:     strings.TrimSpace(member.CharName),
	}
}

func applyLocalGuildPositions(ctx client.Context, positions []network.GuildPosition) {
	if ctx.Session == nil {
		return
	}
	for _, position := range positions {
		index := guildPositionIndex(ctx.Session.Guild.Positions, position.PositionID)
		if index < 0 {
			ctx.Session.Guild.Positions = append(ctx.Session.Guild.Positions, session.GuildPosition{
				PositionID: position.PositionID,
				Right:      position.Right,
				Ranking:    position.Ranking,
				PayRate:    position.PayRate,
			})
			continue
		}
		ctx.Session.Guild.Positions[index].Right = position.Right
		ctx.Session.Guild.Positions[index].Ranking = position.Ranking
		ctx.Session.Guild.Positions[index].PayRate = position.PayRate
	}
	glog.Debugf("guild positions received positions=%d", len(positions))
}

func applyLocalGuildPositionNames(ctx client.Context, positions []network.GuildPosition) {
	if ctx.Session == nil {
		return
	}
	for _, position := range positions {
		index := guildPositionIndex(ctx.Session.Guild.Positions, position.PositionID)
		if index < 0 {
			ctx.Session.Guild.Positions = append(ctx.Session.Guild.Positions, session.GuildPosition{
				PositionID: position.PositionID,
				PosName:    strings.TrimSpace(position.PosName),
			})
			continue
		}
		ctx.Session.Guild.Positions[index].PosName = strings.TrimSpace(position.PosName)
	}
	glog.Debugf("guild position names received positions=%d", len(positions))
}

func applyLocalGuildSkills(ctx client.Context, info network.GuildSkillInfo) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Guild.SkillPoints = info.SkillPoints
	ctx.Session.Guild.Skills = make([]session.Skill, 0, len(info.Skills))
	for _, skill := range info.Skills {
		ctx.Session.Guild.Skills = append(ctx.Session.Guild.Skills, sessionSkillFromNetworkWithResources(ctx.Resources, skill))
	}
	glog.Debugf("guild skill list received count=%d points=%d", len(info.Skills), info.SkillPoints)
}

func applyLocalGuildExpelHistory(ctx client.Context, history []network.GuildExpelHistory) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Guild.ExpelHistory = make([]session.GuildExpelHistory, 0, len(history))
	for _, entry := range history {
		ctx.Session.Guild.ExpelHistory = append(ctx.Session.Guild.ExpelHistory, session.GuildExpelHistory{
			CharName: strings.TrimSpace(entry.CharName),
			Account:  strings.TrimSpace(entry.Account),
			Reason:   strings.TrimSpace(entry.Reason),
		})
	}
	glog.Debugf("guild expel history received entries=%d", len(history))
}

func applyLocalGuildNotice(ctx client.Context, notice network.GuildNotice) {
	if ctx.Session == nil {
		return
	}
	ctx.Session.Guild.NoticeSubject = strings.TrimSpace(notice.Subject)
	ctx.Session.Guild.Notice = strings.TrimSpace(notice.Notice)
	glog.Debugf("guild notice received subject=%q", ctx.Session.Guild.NoticeSubject)
}

func guildPositionIndex(positions []session.GuildPosition, id uint32) int {
	for i, position := range positions {
		if position.PositionID == id {
			return i
		}
	}
	return -1
}

func applyLocalGuildInfo(ctx client.Context, guildID, emblemVersion uint32, name string) {
	name = strings.TrimSpace(name)
	if name == "" && guildID == 0 && emblemVersion == 0 {
		clearPendingGuildName(ctx)
		return
	}
	if ctx.Session != nil {
		if guildID != 0 {
			ctx.Session.GuildID = guildID
		}
		if emblemVersion != 0 {
			ctx.Session.EmblemVersion = emblemVersion
		}
		if name != "" {
			ctx.Session.GuildName = name
		}
		if guildID != 0 {
			ctx.Session.Guild.ID = guildID
		}
		if emblemVersion != 0 {
			ctx.Session.Guild.EmblemVersion = emblemVersion
		}
		if name != "" {
			ctx.Session.Guild.Name = name
		}
		ctx.Session.PendingGuildName = ""
	}
	if ctx.World != nil {
		if guildID != 0 {
			ctx.World.Player.GuildID = guildID
		}
		if emblemVersion != 0 {
			ctx.World.Player.EmblemVersion = emblemVersion
		}
		if name != "" {
			ctx.World.Player.GuildName = name
		}
	}
}
