package bot

import (
	"log/slog"
	"time"

	"github.com/denverquane/slickshift/shift"
	"github.com/denverquane/slickshift/store"
)

func (bot *Bot) StartUserRedemptionProcessing(interval time.Duration, stop <-chan bool) {
	ticker := time.NewTicker(interval)

	for {
		select {
		case <-stop:
			ticker.Stop()
			slog.Info("User code redemption processing stopped")
			return

			// TODO add debouncing so we don't constantly trigger reprocessing if multiple codes come through close together
		case userID := <-bot.redemptionTrigger:
			// if we aren't triggering the code redemption processing for a specific user, then reset the top
			// control flow's interval so we don't run it back-to-back for all users
			if userID == "" {
				ticker.Reset(interval)
			}
			slog.Info("Started user code redemption processing from external trigger")
			bot.userRedemptionLoop(userID)

		case <-ticker.C:
			slog.Info("Started user code redemption processing")
			bot.userRedemptionLoop("")
		}
	}
}

func (bot *Bot) userRedemptionLoop(userID string) {
	var userCookies []store.UserCookies
	var err error
	// if a userID was provided, only get the cookies for that user
	if userID != "" {
		cookies, err := bot.storage.GetDecryptedUserCookies(userID)
		if err != nil {
			slog.Error("Failed to get cookies for user", "user_id", userID, "error", err.Error())
			return
		}
		userCookies = []store.UserCookies{
			store.UserCookies{
				UserID:  userID,
				Cookies: cookies,
			},
		}
		slog.Info("Retrieved decrypted user cookies for specific user", "user_id", userID)
	} else {
		userCookies, err = bot.storage.GetAllDecryptedUserCookiesSorted(-1)
		if err != nil {
			slog.Error("Error getting cookies", "error", err.Error())
			return
		}
		slog.Info("Retrieved decrypted user cookies", "count", len(userCookies))
	}

	for _, user := range userCookies {
		platform, dm, err := bot.storage.GetUserPlatformAndDM(user.UserID)
		if err != nil {
			slog.Error("Error getting platform", "user_id", user.UserID, "error", err.Error())
			continue
		}
		if platform == "" {
			slog.Debug("Skipping user with no platform set", "user_id", user.UserID)
			continue
		}
		codes, err := bot.storage.GetValidCodesNotRedeemedForUser(user.UserID, platform)
		if err != nil {
			slog.Error("Error getting codes", "error", err.Error())
			continue
		}
		slog.Debug("Retrieved unredeemed codes", "user_id", user.UserID, "codes", len(codes))

		client, err := shift.NewClient(user.Cookies)
		if err != nil {
			slog.Error("Error creating shift client", "user_id", user.UserID, "error", err.Error())
			continue
		}

		for _, code := range codes {
			reward, status, err := bot.redeemCode(client, user, code, shift.Platform(platform))
			success := status == shift.SUCCESS
			if err != nil {
				slog.Error("Error redeeming code", "user_id", user.UserID, "code", code, "platform", platform, "error", err.Error())
			} else if reward != nil {
				set, err := bot.storage.SetCodeRewardAndSuccess(code, reward.Title, success)
				if err != nil {
					slog.Error("Error setting code reward", "code", code, "reward", reward.Title, "error", err.Error())
				} else if set {
					slog.Info("Set reward", "code", code, "reward", reward.Title)
				}
			}
			if success && dm {
				str := Cheer + " I successfully redeemed `" + code + "` for you! " + Cheer + "\n\n"
				if reward != nil {
					str += "Looks like the prize was: `" + reward.Title + "`\n"
				}
				err = bot.DMUser(user.UserID, str)
				if err != nil {
					slog.Error("Error DMing user", "user_id", user.UserID, "error", err.Error())
				} else {
					slog.Debug("DMed user", "user_id", user.UserID)
				}
			}
		}
	}
}

// redeemCode redeems a code for a user, and attempts to determine what "reward" was indicated by the redemption
func (bot *Bot) redeemCode(client *shift.Client, user store.UserCookies, code string, platform shift.Platform) (reward *shift.Reward, status string, err error) {
	rewards, err := client.CheckRewards(platform, shift.Borderlands4, -1)
	if err != nil {
		return nil, "", err
	}

	status, err = client.RedeemCode(code, platform)
	if err != nil {
		newRewards, err2 := client.CheckRewards(platform, shift.Borderlands4, -1)
		if err2 != nil {
			return nil, status, err2
		}

		// NOTE: if code redemption starts being performed in parallel for a given user, then this would be unsafe...

		if len(newRewards) > len(rewards) {
			slog.Info("Code redemption returned error, but reward length increased, so presumably it was successful", "error", err.Error())
			reward = &newRewards[0]
			// fallthrough to below, where we add the redemption if it seems successful
		} else {
			return nil, status, err
		}
	}

	// only check the reward if we successfully redeemed. Code above handles if we got an error response, but the rewards increased
	if status == shift.SUCCESS {
		newRewards, err2 := client.CheckRewards(platform, shift.Borderlands4, 1)
		if err2 != nil {
			return nil, status, err2
		}
		reward = &newRewards[0]
	}

	err = bot.storage.AddRedemption(user.UserID, code, string(platform), status)
	if err != nil {
		slog.Error("Error adding redemption", "user_id", user.UserID, "code", code, "platform", platform, "status", status, "error", err.Error())
	} else {
		slog.Info("processed code", "user_id", user.UserID, "code", code, "status", status)
	}
	return reward, status, nil
}
