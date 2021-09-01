package commands

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/zneix/tcb2/internal/bot"
	"go.mongodb.org/mongo-driver/bson"
)

func Subscribed(tcb *bot.Bot) *bot.Command {
	return &bot.Command{
		Name:            "subscribed",
		Aliases:         []string{"tcbsubscribed"},
		Description:     "Shows you list of events you're subscribed to",
		Usage:           "",
		CooldownChannel: 3 * time.Second,
		CooldownUser:    5 * time.Second,
		Run: func(msg twitch.PrivateMessage, args []string) {
			channel := tcb.Channels[msg.RoomID]

			//
			cur, err := tcb.Mongo.CollectionSubs(msg.RoomID).Find(context.TODO(), bson.M{
				"user_id": msg.User.ID,
			})
			if err != nil {
				log.Printf("[Mongo] Failed querying events: " + err.Error())
				return
			}

			subs := []*bot.SubEventSubscription{}

			// Fetch all relevant subscriptions
			for cur.Next(context.TODO()) {
				// Deserialize sub data
				var sub *bot.SubEventSubscription
				err := cur.Decode(&sub)
				if err != nil {
					log.Println("[Mongo] Malformed subscription document: " + err.Error())
					continue
				}
				subs = append(subs, sub)
			}

			// User isn't subscribed to anything, tell them how can they do that
			if len(subs) == 0 {
				// @zneix, You are not subscribed to any events. Use !notifyme <event> [optional value] to subscribe. Valid events are: game, live, offline, title
				eventStrings := []string{}
				for i, desc := range bot.SubEventDescriptions {
					eventStrings = append(eventStrings, fmt.Sprintf("%s (%s)", bot.SubEventType(i), desc))
				}
				channel.Send(fmt.Sprintf("@%s, you are not subscribed to any events. Use %s to subscribe to an event. Valid events: %s", msg.User.Name, "TODO: notifyme", strings.Join(eventStrings, ", ")))
				return
			}

			// Inform the user about their subscriptions
			channel.Send(fmt.Sprintf("@%s, you're subscribed to %d event(s): (TODO: List subscription details)", msg.User.Name, len(subs)))
		},
	}
}
