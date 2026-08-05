package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"os"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

type blockState struct {
	Name       string         `nbt:"name"`
	Properties map[string]any `nbt:"states"`
	Version    int32          `nbt:"version"`
}

func main() {
	input := flag.String("input", "block_palette.nbt", "Cloudburst block_palette.nbt input")
	output := flag.String("output", "block_states.nbt", "Dragonfly block_states.nbt output")
	flag.Parse()

	in, err := os.Open(*input)
	check(err)
	decompressed, err := gzip.NewReader(in)
	check(err)

	var palette map[string]any
	check(nbt.NewDecoderWithEncoding(decompressed, nbt.BigEndian).Decode(&palette))
	check(decompressed.Close())
	check(in.Close())

	out, err := os.Create(*output)
	check(err)
	encoder := nbt.NewEncoder(out)
	blocks, ok := palette["blocks"].([]any)
	if !ok {
		panic(fmt.Errorf("blocks has type %T, expected []any", palette["blocks"]))
	}
	for index, raw := range blocks {
		entry, ok := raw.(map[string]any)
		if !ok {
			panic(fmt.Errorf("block %d has type %T, expected map[string]any", index, raw))
		}
		state := blockState{
			Name:       value[string](entry, "name", index),
			Properties: value[map[string]any](entry, "states", index),
			Version:    value[int32](entry, "version", index),
		}
		check(encoder.Encode(state))
	}
	check(out.Close())
	fmt.Printf("Wrote %d block states to %s\n", len(blocks), *output)
}

func value[T any](entry map[string]any, key string, index int) T {
	result, ok := entry[key].(T)
	if !ok {
		panic(fmt.Errorf("block %d field %q has type %T", index, key, entry[key]))
	}
	return result
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
