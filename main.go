package main

import (
	pokechache "awesomeProject1/internal"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

type cliCommand struct {
	name     string
	desc     string
	callback func(*config, string) error
}

type config struct {
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`

	Cache   *pokechache.Cache
	Pokedex map[string]*Pokemon
}

type Pokemon struct {
	Height int `json:"height"`
	Stats  []struct {
		BaseStat int `json:"base_stat"`
		Effort   int `json:"effort"`
		Stat     struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Slot int `json:"slot"`
		Type struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"type"`
	} `json:"types"`
	Weight int `json:"weight"`
}

type pokeInfo struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

func main() {
	cfg := &config{
		Next:    "https://pokeapi.co/api/v2/location-area?offset=0&limit=20",
		Cache:   pokechache.NewCache(10 * time.Second),
		Pokedex: make(map[string]*Pokemon),
	}
	startRepl(cfg)
}

func startRepl(cfg *config) {
	cliCommands := getCliCommands()
	for {
		fmt.Print("pokedex > ")
		buf := bufio.NewScanner(os.Stdin)
		buf.Scan()
		err := buf.Err()
		if err != nil {
			log.Fatal(err)
		}
		line := buf.Text()
		arguments := strings.Split(line, " ")
		areaName := ""
		if len(arguments) == 2 {
			areaName = arguments[1]
		}
		command, ok := cliCommands[arguments[0]]
		fmt.Println(line)
		if ok {
			err = command.callback(cfg, areaName)
			if err != nil {
				log.Fatal(err)
			}
			continue
		} else {
			fmt.Println("Command not found")
			continue
		}
	}
}

func getCliCommands() map[string]cliCommand {
	return map[string]cliCommand{
		"help": {
			name:     "help",
			desc:     "displays the names of next 20 location areas",
			callback: commandHelp,
		},
		"map": {
			name:     "map",
			desc:     "displays the previous 20 location areas",
			callback: commandMap,
		},
		"mapb": {
			name:     "mapb",
			desc:     "Displays a map message",
			callback: commandMapBack,
		},
		"explore": {
			name:     "explore",
			desc:     "see a list of all the Pokémon in a given area",
			callback: commandExplore,
		},
		"catch": {
			name:     "catch",
			desc:     "Catching Pokemon adds them to the user's Pokedex",
			callback: commandCatch,
		},
		"inspect": {
			name:     "inspect",
			desc:     "Displays information about the current Pokemon",
			callback: commandInspect,
		},

		"pokedex": {
			name:     "pokedex",
			desc:     "print a list of all the names of the Pokemon the user has caught",
			callback: commandPokedex,
		},

		"exit": {
			name:     "exit",
			desc:     "Exit the Pokedex",
			callback: commandExit,
		},
	}
}

func commandPokedex(cfg *config, _ string) error {
	for key := range cfg.Pokedex {
		fmt.Println("-", key)
	}
	return nil
}

func commandInspect(cfg *config, pokemonName string) error {
	val, ok := cfg.Pokedex[pokemonName]
	if !ok {
		return errors.New("You have not caught " + pokemonName)
	}

	fmt.Printf("name: %s\nheight: %v\nweight:%v", pokemonName, val.Height, val.Weight)

	fmt.Println("Stats:")
	for _, stats := range val.Stats {
		fmt.Printf("-%s: %v\n", stats.Stat.Name, stats.BaseStat)
	}

	fmt.Println("Type(s):")
	for _, types := range val.Types {
		fmt.Printf("-%s\n", types.Type.Name)
	}

	return nil
}

func commandCatch(cfg *config, pokemonName string) error {
	chance := rand.Intn(100)
	fmt.Println("Throwing a Pokeball at " + pokemonName + "...")
	if chance < 60 {
		fmt.Println(pokemonName + " escaped")
		return nil
	}
	fmt.Println(pokemonName + " caught")
	data, err := getPokemon(pokemonName)

	if err != nil {
		return err
	}
	cfg.Pokedex[pokemonName] = &Pokemon{}
	err = processPokemonData(cfg, pokemonName, data)

	if err != nil {
		return err
	}

	return nil
}

func processPokemonData(cfg *config, pokemonName, data string) error {
	err := json.Unmarshal([]byte(data), cfg.Pokedex[pokemonName])

	if err != nil {
		return err
	}
	return nil
}

func getPokemon(pokemonName string) (string, error) {
	resp, err := http.Get("https://pokeapi.co/api/v2/pokemon/" + pokemonName)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode > 299 {
		return "", errors.New(resp.Status)
	}
	if err != nil {
		return "", err
	}
	return string(body), nil

}

func commandExplore(cfg *config, areaName string) error {

	data, err := getPokemonsByArea(cfg, areaName)
	if err != nil {
		log.Fatal(err)
	}
	pokemonNames, err := processPokemonByAreaData(cfg, data)
	if err != nil {
		log.Fatal(err)
	}

	for _, pokemonName := range pokemonNames {
		fmt.Println(pokemonName)
	}

	return nil
}

func getPokemonsByArea(cfg *config, areaName string) (string, error) {
	resp, err := http.Get("https://pokeapi.co/api/v2/location-area/" + areaName + "/")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode > 299 {
		return "", errors.New(resp.Status)
	}
	return string(body), nil
}

func commandMap(cfg *config, areaName string) error {
	data, _ := getAreas(cfg.Next)

	cfg.Cache.Add(cfg.Next, []byte(data))

	areaNames, err := processAreaData(cfg, data)
	if err != nil {
		return err
	}
	for _, name := range areaNames {
		fmt.Println(name)
	}
	return nil
}

func commandMapBack(cfg *config, areaName string) error {
	if cfg.Previous == "" {
		return errors.New("no previous location found")
	}
	data := ""
	if val, ok := cfg.Cache.Get(cfg.Previous); ok {
		data = string(val)
	} else {
		data, _ = getAreas(cfg.Previous)
	}

	areaNames, err := processAreaData(cfg, data)
	if err != nil {
		return err
	}
	for _, name := range areaNames {
		fmt.Println(name)
	}
	return nil
}

func processPokemonByAreaData(cfg *config, data string) ([]string, error) {
	pokeData := pokeInfo{}
	processedData := make([]string, 0)
	err := json.Unmarshal([]byte(data), &pokeData)
	if err != nil {
		return nil, err
	}

	for _, result := range pokeData.PokemonEncounters {
		processedData = append(processedData, result.Pokemon.Name)
	}

	return processedData, nil

}

func processAreaData(cfg *config, data string) ([]string, error) {
	processedData := make([]string, 0)
	err := json.Unmarshal([]byte(data), cfg)
	if err != nil {
		return nil, err
	}

	for _, areaData := range cfg.Results {
		processedData = append(processedData, areaData.Name)
	}

	return processedData, nil
}

func getAreas(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		return "", errors.New(resp.Status)
	}
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func commandHelp(cfg *config, _ string) error {
	fmt.Println("Welcome to the Pokedex!\n" +
		"Available commands:\n" +
		"help: Displays this help message\n" +
		"map: Displays the next 20 location areas\n" +
		"mapb: Displays the previous 20 location areas\n" +
		"explore <area>: Displays a list of all the Pokémon in a given area\n" +
		"catch <pokemon>: Attempts to catch a Pokémon by name\n" +
		"inspect <pokemon>:Shows info about a specific pokemon\n" +
		"exit: Exits the Pokedex")
	return nil
}

func commandExit(cfg *config, areaName string) error {
	fmt.Println("Bye!")
	defer os.Exit(0)
	return nil
}
