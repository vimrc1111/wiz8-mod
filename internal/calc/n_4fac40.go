package calc

// The fixture uses eight party records, one monster/group and an empty scene
// event list. Source descriptors are created by 0x53be00, queued targets are
// at character subrecord +0x1d (stride 0x106), and character stride is 0x1862.
// UI messages do not affect the returned byte; their arena strings are empty.
func spellCastEligibilityNode(in []int64, r Rand) []int64 {
	if len(in) != len(spellCastEligibilityInputs) {
		panic("spell eligibility: input arity")
	}
	v := make(map[string]int64, len(in))
	for i, name := range spellCastEligibilityInputs {
		v[name] = in[i]
	}
	area := make([]int64, len(spellAreaInputs))
	for i, name := range spellAreaInputs {
		area[i] = v[name]
	}
	set := func(name string, value int64) {
		for i, n := range spellAreaInputs {
			if n == name {
				area[i] = value
				return
			}
		}
		panic(name)
	}
	set("sourceKind", 1)
	set("sourceCharacter", v["caster"])
	set("sourceMonster", -1)
	set("battleObject", 1)
	set("battleSuppressFallback", v["partyCastFlag"])
	// Both local lists start empty with capacity five (0x4fac61..0x4faca3).
	set("monsterCapacity", 5)
	set("characterCapacity", 5)
	targets := spellAreaTargetSelectionNode(area, r)
	// Out of combat, 0x51b4e0 validates the queued target first. This
	// monster has no party-target action (+0x34c at 0x519f9d), and the
	// group has no usable member action (0x5194a1 -> 0x537780). An invalid
	// source-target pair makes 0x4fd030 return before populating either list.
	if v["inCombat"] == 0 && (v["targetKind"] == 3 || v["targetKind"] == 4) {
		targets[0], targets[1] = 0, 0
	}
	eligible := int64(1)
	id, typ := v["spellID"], int64(int32(v["spellTargetType"]))
	// ID 30 and target modes 8..10 can cast without a nonempty hit list;
	// the literal boundaries are from 0x4facff and 0x4fad1a..0x4fad22.
	if id != 30 && targets[0] == 0 && targets[1] == 0 && (typ < 8 || typ > 10) {
		eligible = 0
	}
	// The event list is empty in this arena, so 0x4d9080 returns false
	// whether event scanning is enabled or suppressed by 0x68f105.
	// ID 75 requires scene bit 4 and forbids bits 0 and 9
	// (0x4fada1, 0x41f070, 0x41eff0).
	flags := uint32(v["sceneFlags"])
	if id == 75 && (flags&1 != 0 || flags&16 == 0 || flags&512 != 0) {
		eligible = 0
	}
	return []int64{eligible}
}

var spellCastEligibilityInputs = []string{"targetXBits", "targetYBits", "targetZBits", "partyXBits", "partyYBits", "partyZBits", "partyAimXBits", "partyAimYBits", "partyAimZBits", "partyRadiusBits", "spellID", "spellTargetType", "areaBaseBits", "areaPerPowerBits", "spellRange", "powerLevel", "targetKind", "targetCharacter", "targetMonster", "targetGroup", "monsterPresent", "monsterVisible", "monsterFaction", "monsterHP", "monsterRemoved", "monsterCondition", "monsterMagicMouth", "monsterPartySight2", "monsterPartySight3", "monsterImmune", "monsterSpecies", "monsterSpecialStatus", "monsterXBits", "monsterYBits", "monsterZBits", "monsterRadiusBits", "monsterEyeHeightBits", "monsterSpellReachBits", "partySpellReachBits", "groupFaction", "groupMemberCount", "character0present", "character0HP", "character0condition", "character0magicProtection", "character1present", "character1HP", "character1condition", "character1magicProtection", "character2present", "character2HP", "character2condition", "character2magicProtection", "character3present", "character3HP", "character3condition", "character3magicProtection", "character4present", "character4HP", "character4condition", "character4magicProtection", "character5present", "character5HP", "character5condition", "character5magicProtection", "character6present", "character6HP", "character6condition", "character6magicProtection", "character7present", "character7HP", "character7condition", "character7magicProtection", "monsterRayVisible", "monsterModelScaleBits", "monsterBoundsMinYBits", "monsterBoundsMaxYBits", "caster", "alternateMessage", "skipSceneEvent", "sceneFlags", "inCombat", "combatPendingCount", "partyCastFlag", "sceneEventsDisabled"}

func init() {
	Nodes["004fac40"] = Node{Addr: "004fac40", In: spellCastEligibilityInputs, Out: []string{"eligible"}, F: spellCastEligibilityNode}
}
