package pocketmine

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/datagen/data"
	"github.com/df-mc/datagen/write"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func HandleGameData(gameData minecraft.GameData) {
	requiredItemList := make(map[string]RequiredItemEntry)
	for _, item := range gameData.Items {
		data.ItemNameToNetworkID[item.Name] = int32(item.RuntimeID)
		data.ItemNetworkIDToName[int32(item.RuntimeID)] = item.Name
		requiredItemList[item.Name] = RequiredItemEntry{
			RuntimeID:      item.RuntimeID,
			ComponentBased: item.ComponentBased,
		}
	}
	write.JSON("output/pocketmine/required_item_list.json", requiredItemList)
}

func HandleAvailableActorIdentifiers(pk *packet.AvailableActorIdentifiers) {
	var identifiers AvailableActorIdentifiers
	err := nbt.Unmarshal(pk.SerialisedEntityIdentifiers, &identifiers)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal entity identifiers: %w", err))
	}
	list := identifiers.IDList
	slices.SortFunc(list, func(a, b ActorIdentifier) int {
		return int(a.RuntimeID - b.RuntimeID)
	})
	var lines []string
	for _, id := range list {
		lines = append(lines, fmt.Sprintf("\t\"%s\": %d", id.ID, id.RuntimeID))
	}
	b := []byte(fmt.Sprintf("{\n%s\n}", strings.Join(lines, ",\n")))
	write.Raw("output/pocketmine/entity_id_map.json", b)
	write.Raw("output/pocketmine/entity_identifiers.nbt", pk.SerialisedEntityIdentifiers)
}

func HandleBiomeDefinitionList(pk *packet.BiomeDefinitionList) {
	biomes := make(map[string]BiomeDefinition)
	list := pk.StringList
	for _, definition := range pk.BiomeDefinitions {
		name := list[definition.NameIndex]
		biomes[name] = newBiomeDefinition(definition, list)
	}
	write.JSON("output/pocketmine/biome_definitions.json", biomes)
}

func HandleCraftingData(pk *packet.CraftingData) {
	recipes := make(map[string][]any)
	for i := range pk.ShapelessRecipes {
		recipes["shapeless_crafting"] = append(recipes["shapeless_crafting"], shapelessRecipeData(&pk.ShapelessRecipes[i]))
	}
	for i := range pk.ShapedRecipes {
		key := "shaped_crafting"
		if !pk.ShapedRecipes[i].AssumeSymmetry {
			key += "_asymmetric"
		}
		recipes[key] = append(recipes[key], shapedRecipeData(&pk.ShapedRecipes[i]))
	}
	for i := range pk.MultiRecipes {
		recipes["special_hardcoded"] = append(recipes["special_hardcoded"], pk.MultiRecipes[i].UUID.String())
	}
	for i := range pk.ShulkerBoxRecipes {
		recipes["shapeless_shulker_box"] = append(recipes["shapeless_shulker_box"], shapelessRecipeData(&pk.ShulkerBoxRecipes[i].ShapelessRecipe))
	}
	for i := range pk.ShapelessChemistryRecipes {
		recipes["shapeless_chemistry"] = append(recipes["shapeless_chemistry"], shapelessRecipeData(&pk.ShapelessChemistryRecipes[i].ShapelessRecipe))
	}
	for i := range pk.ShapedChemistryRecipes {
		key := "shaped_chemistry"
		if !pk.ShapedChemistryRecipes[i].AssumeSymmetry {
			key += "_asymmetric"
		}
		recipes[key] = append(recipes[key], shapedRecipeData(&pk.ShapedChemistryRecipes[i].ShapedRecipe))
	}
	for i := range pk.SmithingTransformRecipes {
		recipes["smithing"] = append(recipes["smithing"], smithingTransformRecipeData(&pk.SmithingTransformRecipes[i]))
	}
	for i := range pk.SmithingTrimRecipes {
		recipes["smithing_trim"] = append(recipes["smithing_trim"], smithingTrimRecipeData(&pk.SmithingTrimRecipes[i]))
	}
	for _, r := range pk.PotionRecipes {
		recipes["potion_type"] = append(recipes["potion_type"], potionTypeRecipeData(r))
	}
	for _, r := range pk.PotionContainerChangeRecipes {
		recipes["potion_container_change"] = append(recipes["potion_container_change"], potionContainerChangeRecipeData(r))
	}

	type keyValue struct {
		k string
		v any
	}
	for name, entries := range recipes {
		var sorted []keyValue
		seen := make(map[string]int)
		for _, entry := range entries {
			data, _ := json.Marshal(entry)
			key := string(data)
			dupe, _ := seen[key]
			seen[key] = dupe + 1
			suffix := string('a' + rune(dupe))
			sorted = append(sorted, keyValue{key + suffix, entry})
		}
		slices.SortFunc(sorted, func(a, b keyValue) int {
			return strings.Compare(a.k, b.k)
		})
		recipes[name] = mapSlice(sorted, func(kv keyValue) any {
			return kv.v
		})
		for key, count := range seen {
			if count > 1 {
				fmt.Printf("warning: %s recipe %s was seen %d times\n", name, key, count)
			}
		}
	}
	for k, v := range recipes {
		write.JSON(fmt.Sprintf("output/pocketmine/recipes/%s.json", k), v)
	}
}

func HandleCreativeContent(pk *packet.CreativeContent) {
	var content CreativeItems
	for _, group := range pk.Groups {
		content.Groups = append(content.Groups, CreativeGroup{
			CategoryID:   int32(group.Category),
			CategoryName: group.Name,
			Icon:         itemStackData(group.Icon),
		})
	}
	for _, item := range pk.Items {
		content.Items = append(content.Items, CreativeItem{
			GroupID: item.GroupIndex,
			Item:    itemStackData(item.Item),
		})
	}
	write.JSON("output/pocketmine/creativeitems.json", content)
}

func mapSlice[A, B any](s []A, f func(A) B) []B {
	var r []B
	for _, v := range s {
		r = append(r, f(v))
	}
	return r
}
