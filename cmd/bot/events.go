package main

import (
	"log"
	"strings"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/nicklaw5/helix"
	"github.com/zneix/tcb2/internal/bot"
)

func registerEvents(tcb *bot.Bot) {
	// Twitch IRC events

	// Authenticated with IRC
	tcb.TwitchIRC.OnConnect(func() {
		log.Println("[TwitchIRC] connected")
		joinChannels(tcb)
	})

	// PRIVMSG
	tcb.TwitchIRC.OnPrivateMessage(func(message twitch.PrivateMessage) {
		// Ignore non-commands
		if !strings.HasPrefix(message.Message, COMMANDPREFIX) {
			return
		}

		// Parse command name and arguments
		args := strings.Fields(message.Message)
		commandName := args[0][len(COMMANDPREFIX):]
		args = args[1:]

		// Try to find the command by its name and/or aliases
		command, exists := tcb.Commands.GetCommand(commandName)
		if !exists {
			return
		}

		// TODO: [Permissions] Check if user is allowed to execute the command

		// Check if channel or user is on cooldown
		if time.Since(command.LastExecutionChannel[message.RoomID]) < command.CooldownChannel || time.Since(command.LastExecutionUser[message.User.ID]) < command.CooldownUser {
			return
		}

		// Execute the command
		command.Run(message, args)

		// Apply cooldown if user's permissions don't allow to skip it
		// TODO: [Permissions] Don't apply user cooldowns to users that are allowed to skip it
		command.LastExecutionChannel[message.RoomID] = time.Now()
		command.LastExecutionUser[message.User.ID] = time.Now()
	})

	// USERSTATE
	tcb.TwitchIRC.OnUserStateMessage(func(message twitch.UserStateMessage) {
		channelID, ok := tcb.Logins[message.Channel]
		if !ok {
			// tcb.Logins map didn't have current channel's ID
			// Note: this should realistically never occur though, but early exit to prevent panic
			return
		}

		channel := tcb.Channels[channelID]

		// Check if Channel.Mode changed by comparing bot's state
		newMode := bot.ChannelModeNormal

		// Bot will always have elevated permissions in its own chat, saving some time with the early-out
		if channel.Login == tcb.Self.Login {
			return
		}

		userType, ok := message.Tags["user-type"]
		switch {
		case !ok:
			log.Println("[USERSTATE] user-type tag was not found in the IRC message, either no capabilities or Twitch removed this tag xd")

		case userType == "mod":
			newMode = bot.ChannelModeModerator

		default:
			// Since user-type does not care about VIP status, we need to check badges
			for key := range message.User.Badges {
				if key == "vip" || key == "moderator" {
					newMode = bot.ChannelModeModerator
					break
				}
			}
		}

		// Update ChannelMode in the current channel if it differs
		if newMode != channel.Mode {
			err := channel.ChangeMode(tcb.Mongo, newMode)
			if err != nil {
				log.Printf("Failed to change mode in %s: %s\n", channel, err)
			}
		}
	})

	// NOTICE
	tcb.TwitchIRC.OnNoticeMessage(func(message twitch.NoticeMessage) {
		channelID, ok := tcb.Logins[message.Channel]
		if !ok {
			// tcb.Logins map didn't have current channel's ID
			// Note: this should realistically never occur though, but early exit to prevent panic
			return
		}
		channel := tcb.Channels[channelID]

		log.Printf("[TwitchIRC:NOTICE] %s in %s\n", message.MsgID, channel)

		switch message.MsgID {
		case "msg_banned", "msg_channel_suspended":
			err := channel.ChangeMode(tcb.Mongo, bot.ChannelModeInactive)
			if err != nil {
				log.Printf("Failed to change mode in %s: %s\n", channel, err)
			}
		default:
		}
	})

	// Twitch EventSub events

	// channel.update
	tcb.EventSub.OnChannelUpdateEvent(func(event helix.EventSubChannelUpdateEvent) {
		// TODO: Handle received event
		log.Printf("[EventSub:channel.update] %# v\n", event)
	})

	// stream.online
	tcb.EventSub.OnStreamOnlineEvent(func(event helix.EventSubStreamOnlineEvent) {
		// TODO: Handle received event
		log.Printf("[EventSub:stream.online] %# v\n", event)
	})

	// stream.offline
	tcb.EventSub.OnStreamOfflineEvent(func(event helix.EventSubStreamOfflineEvent) {
		// TODO: Handle received event
		log.Printf("[EventSub:stream.offline] %# v\n", event)
	})
}
