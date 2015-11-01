# go-bus

`go-bus` is a plain event bus for Go, allowing otherwise decoupled
components to publish and listen for events.

## Usage

A simple publish onto the default bus:

	bus.SubscribeFunc("greetings", func(b *Bus, topic, value interface{}) {
		log.Printf("Received greeting: %v", value)
	})
	bus.Publish("greetings", "Good morning Mr Freeman")

Using a custom bus and deregistering a subscriber after use:

	b := bus.NewBus()

	// Log all kills and deaths to console
	eventLogger := func(b *Bus, t, v interface{}) {
		switch ev := v.(type) {
			case Kill:
				log.Printf("You killed %s", ev.Victim)
			case Join:
				log.Printf("%s joined", ev.Player)
		}
	}
	b.SubscribeFunc("kills", eventLogger)
	b.SubscribeFunc("joins", eventLogger)

	// Set player flag when Breen is killed
	dereg := b.SubscribeFunc("kills", func(b *Bus, t, v interface{}) {
		kill := v.(Kill)
		if kill.Victim == "Breen" {
			flags.KilledBreen = true
			dereg() // stop listening for events
		}
	})

	// Publish events
	b.Publish("kills", Kill{Victim: "Vortigaunt"})
	b.Publish("kills", Kill{Victim: "Breen"})

Sending events asynchronously using a non-blocking `Publish` call via the `Async` flag:

	// Listen for releases, and start download immediately
	bus.SubscribeFunc("release", func(b *Bus, t, v interface{}) {
		if err := download(v); err != nil {
			log.Printf("download failed: %v", err)
		}
		log.Printf("%v download complete!", v)
	})

	// Book holiday for certain releases.
	bus.SubscribeFunc("games", func(b *Bus, t, v interface{}) {
		if v == "hl3" {
			bookHoliday(5 * 24 * time.Hour)
		}
	})

	for releases := range releaseChan {
		// Call asynchronously as handlers may take some time...
		bus.Publish("games", release, Async)
	}


