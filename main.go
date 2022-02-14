package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	err := mainReturnWithCode()
	if err == nil {
		os.Exit(0)
	}
	log.Fatal(err)
	os.Exit(-1)
}

type key_t struct {
	Index   int
	KeyText string
	Keycode string
}

type layer_t struct {
	Name     string `json:"name"`
	Filename string `json:"layer"`
	Lines    [][]rune
	Keys     []key_t
}

type keymap_t struct {
	NumberOfKeys    int               `json:"number_of_keys"`
	SymbolToKeycode map[string]string `json:"symbol_to_keycode"`
	Reference       layer_t           `json:"layer.empty"`
	Template        layer_t           `json:"layer.template"`
	Layers          []layer_t         `json:"layers"`
	Keymap_C_Pre    []string          `json:"keymap.c.pre"`
	Keymap_C_Layer  []string          `json:"keymap.c.layer"`
	Keymap_C_Post   []string          `json:"keymap.c.post"`
	Layers_H_Pre    []string          `json:"layers.h.pre"`
	Layers_H_Post   []string          `json:"layers.h.post"`
}

func readLayerFile(filename string) ([][]rune, error) {
	// open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read each line as an array of runes
	var lines [][]rune
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, []rune(line))
	}

	// return the array of runes
	return lines, nil
}

// read the keymap.json file and return the keymap
func readKeymapFile(filename string) (*keymap_t, error) {
	// open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// read the keymap
	keymap := new(keymap_t)
	err = json.NewDecoder(file).Decode(keymap)
	if err != nil {
		return nil, err
	}

	// return the keymap
	return keymap, nil
}

// filter out characters from a layer that are in the reference layer
func filterCharactersFromLayer(reference, layer [][]rune) ([][]rune, error) {
	// loop through the reference layer
	for i := 0; i < len(reference); i++ {
		// loop through the layer
		for j := 0; j < len(layer); j++ {

			for k := 0; k < len(reference[i]) && k < len(layer[j]); k++ {
				// if the characters match, replace the character in the layer with a space
				if reference[i][k] == layer[j][k] {
					layer[j][k] = ' '
				} else if reference[j][k] != ' ' {
					// if the characters don't match, but the reference character is not a space, return an error
					// that includes the mismatching characters as well as the line and character position
					return nil, fmt.Errorf("character mismatch at line %d, character %d: %c != %c", i, k, reference[i][k], layer[j][k])
				}
			}
		}
	}

	// return the template layer
	return layer, nil
}

//  write a layer to a file
func writeLayerToFile(file *os.File, layer_c_code []string, layer *layer_t) error {

	// replace ${LAYER_NAME} with the layer name in the lines of layer_c_code
	for i := 0; i < len(layer_c_code); i++ {
		layer_c_code[i] = strings.Replace(layer_c_code[i], "${LAYER_NAME}", layer.Name, -1)
	}
	// for each key using the key index replace it in the lines of layer_c_code
	for i := 0; i < len(layer.Keys); i++ {
		key_index := strconv.Itoa(layer.Keys[i].Index)
		// adjust key_index to be the correct length by prepending zeros
		key_index = strings.Repeat("0", 3-len(key_index)) + key_index
		key_str := "____" + key_index + "____"

		for j := 0; j < len(layer_c_code); j++ {
			keycode_str := layer.Keys[i].Keycode

			// adjust keycode_str to be the same length as key_str by prepending spaces
			// do nothing if keycode_str is already the correct length or longer
			if len(keycode_str) < len(key_str) {
				keycode_str = strings.Repeat(" ", len(key_str)-len(keycode_str)) + keycode_str
			}
			layer_c_code[j] = strings.Replace(layer_c_code[j], key_str, keycode_str, -1)
		}
	}

	// write the layer
	for i := 0; i < len(layer_c_code); i++ {
		_, err := file.WriteString(layer_c_code[i] + "\n")
		if err != nil {
			return err
		}
	}

	// return no error
	return nil
}

func mainReturnWithCode() error {

	keymap, err := readKeymapFile("keymap.json")
	if err != nil {
		return err
	}

	keymap.Template.Lines, err = filterCharactersFromLayer(keymap.Reference.Lines, keymap.Template.Lines)
	if err != nil {
		return err
	}

	// initialize each layer
	for l := 0; l < len(keymap.Layers); l++ {
		layer := &keymap.Layers[l]
		layer.Lines, err = readLayerFile(layer.Filename)
		if err != nil {
			return err
		}
		layer.Lines, err = filterCharactersFromLayer(keymap.Reference.Lines, layer.Lines)
		if err != nil {
			return err
		}
		layer.Keys = make([]key_t, keymap.NumberOfKeys)
		// set each key to an invalid state
		for k := 0; k < keymap.NumberOfKeys; k++ {
			layer.Keys[k].Index = -1
		}
	}

	//   scan the template layer and at the first position of a digit scan forward until a non-digit is found
	//   for each layer collect the characters from the same position into a key string
	//   convert the collected digits to a number and use that as the key index
	//   for each layer append the collected string to the key with the key index

	// loop through the template layer
	for i := 0; i < len(keymap.Template.Lines); i++ {
		// loop through the layer
		for j := 0; j < len(keymap.Template.Lines[i]); j++ {
			// if the character is a digit
			if keymap.Template.Lines[i][j] >= '0' && keymap.Template.Lines[i][j] <= '9' {
				// scan forward until a non-digit is found
				for k := j + 1; k < len(keymap.Template.Lines[i]); k++ {
					// if the character is not a digit
					if keymap.Template.Lines[i][k] < '0' || keymap.Template.Lines[i][k] > '9' {
						keyStr := string(keymap.Template.Lines[i][j:k])
						// convert the collected digits to a number and use that as the key index
						keyIndex, err := strconv.Atoi(keyStr)
						if err != nil {
							return err
						}

						// for each layer append the collected string to the key with the key index
						for l := 0; l < len(keymap.Layers); l++ {
							str := string(keymap.Layers[l].Lines[i][j:k])
							// trim any space character from str
							str = strings.Replace(str, " ", "", -1)
							// is this the first time we encounter this key?
							if keymap.Layers[l].Keys[keyIndex].Index == -1 {
								// initialize the key
								key := key_t{
									Index:   keyIndex,
									KeyText: str,
									Keycode: "",
								}
								keymap.Layers[l].Keys[keyIndex] = key
							} else {
								keymap.Layers[l].Keys[keyIndex].KeyText += str
							}
						}

						// move the pointer to the next character
						j = k
						break
					}
				}
			}
		}
	}

	// for each layer
	//   for each key
	//      get the keycode using KeyText and keymap_t.Keys
	for l := 0; l < len(keymap.Layers); l++ {
		layer := &keymap.Layers[l]
		for k := 0; k < len(layer.Keys); k++ {
			key := &layer.Keys[k]
			if key.Index != -1 {
				keycode, exists := keymap.SymbolToKeycode[key.KeyText]
				if !exists {
					return fmt.Errorf("keycode not found for key %s", key.KeyText)
				}
				key.Keycode = keycode
			}
		}
	}

	// write keymap.c
	file, err := os.Create("keymap.c")
	if err != nil {
		return err
	}
	defer file.Close()

	// write the keymap pre part to the file
	for _, line := range keymap.Keymap_C_Pre {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	// write the layers to the file

	for l := 0; l < len(keymap.Layers); l++ {
		layer := &keymap.Layers[l]
		writeLayerToFile(file, keymap.Keymap_C_Layer, layer)
	}

	// write the keymap post part to the file
	for _, line := range keymap.Keymap_C_Post {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	// write layers.h
	file, err = os.Create("layers.h")
	if err != nil {
		return err
	}
	defer file.Close()

	// write the layers pre part to the file
	for _, line := range keymap.Layers_H_Pre {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	// write the layer names as enums to the file
	for l := 0; l < len(keymap.Layers); l++ {
		layer := &keymap.Layers[l]
		_, err = file.WriteString(fmt.Sprintf("    %s = %d\n", layer.Name, l))
		if err != nil {
			return err
		}
	}

	// write the layers post part to the file
	for _, line := range keymap.Layers_H_Post {
		_, err = file.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}
