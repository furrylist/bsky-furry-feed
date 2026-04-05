package feed

import (
	"fmt"
	"strings"
	"time"
)

type conFeed struct {
	ID        string
	Name      string
	MainTags  []string
	OtherTags []string
}

// registerChronologicalConFeed registers a con feed (with con- prefix).
//
// It appends the current year to the feed name but it doesn't generate the
// hashtags based on the year.
func (s *Service) registerChronologicalConFeed(feed conFeed) {
	var tagList []string
	for i, tag := range feed.MainTags {
		if i == len(feed.MainTags)-1 {
			tagList = append(tagList, "or #"+tag)
		} else {
			tagList = append(tagList, "#"+tag)
		}
	}
	var tags string
	if len(tagList) > 2 {
		tags = strings.Join(tagList, ", ")
	} else {
		tags = strings.Join(tagList, " ")
	}
	s.Register(Meta{
		ID:          "con-" + feed.ID,
		DisplayName: fmt.Sprintf("%s %s %d", PawEmoji, feed.Name, time.Now().Year()),
		Description: fmt.Sprintf(
			"A feed for all things %s! Use %s to include a post in the feed.\n\nJoin the furry feeds by following @furryli.st",
			feed.Name,
			tags,
		),
	}, chronologicalGenerator(chronologicalGeneratorOpts{
		generatorOpts: generatorOpts{
			Hashtags:           append(feed.MainTags, feed.OtherTags...),
			DisallowedHashtags: defaultDisallowedHashtags,
		},
	}))
}

func registerConFeeds(r *Service) {
	r.registerChronologicalConFeed(conFeed{
		ID:       "denfur",
		Name:     "DenFur",
		MainTags: []string{"denfur", "denfur2026"},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:   "eurofurence",
		Name: "Eurofurence",
		MainTags: []string{
			"eurofurence", "eurofurence2025", "eurofurence29",
			"ef", "ef2025", "ef29",
		},
		OtherTags: []string{
			"eurofurence2023", "eurofurence27", "ef2023", "ef27",
			"eurofurence2024", "eurofurence28", "ef2024", "ef28",
			"euroference",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "blfc",
		Name:     "BLFC",
		MainTags: []string{"blfc", "blfc26", "blfc2026"},
		OtherTags: []string{
			"blfc25", "blfc2025",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "mff",
		Name:     "MFF",
		MainTags: []string{"furfest", "mff", "mff26", "mff2026"},
		OtherTags: []string{
			"furfest24", "furfest2024", "mff24", "mff2024",
			"furfest25", "furfest2025", "mff25", "mff2025",
			"furfest26", "furfest2026",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "fc",
		Name:     "FC",
		MainTags: []string{"fc", "fc26", "fc2026"},
		OtherTags: []string{
			"furcon26", "furcon2026",
			"furtherconfusion", "furtherconfusion26", "furtherconfusion2026",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "nfc",
		Name:     "NFC",
		MainTags: []string{"nfc", "nfc26", "nfc2026"},
		OtherTags: []string{
			"nordicfuzzcon", "nordicfuzzcon26", "nordicfuzzcon2026",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "fwa",
		Name:     "FWA",
		MainTags: []string{"fwa", "fwa26", "fwa2026", "furryweekend"},
		OtherTags: []string{
			"fwa25", "fwa2025",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "ac",
		Name:     "Anthrocon",
		MainTags: []string{"anthrocon", "anthrocon2026", "ac"},
		OtherTags: []string{
			"anthrocon26", "ac2026", "ac26",
		},
	})
	r.registerChronologicalConFeed(conFeed{
		ID:       "lvfc",
		Name:     "LVFC",
		MainTags: []string{"lvfc", "lvfc26"},
		OtherTags: []string{
			"lasvegasfurcon", "lvfc2026",
		},
	})
}
