## Overview
SlickShift does not store **any** usernames or passwords.

Instead, it uses SHiFT website session cookies, which are encrypted in the Bot's database.  

These cookies are only used to log you into SHiFT Rewards so the bot can redeem codes for you — and nothing else. 

Your data is never shared with third parties, and the entire project is completely open-source, so anyone can review exactly how it works [here on GitHub](https://github.com/denverquane/slickshift) 

### `/login-insecure` vs. `/login` 
If you specify your login details with `/login-insecure`, SlickShift uses your username/password to login to SHiFT on your behalf, obtain session cookies, and then encrypt and store those cookies for later.
You can see the exact process detailed in code [here](./bot/login-insecure.go).

Once this process completes, *your username/password are completely forgotten/discarded by the bot.* 

`/login` takes a different approach, and instead requires users to provide session cookies directly; no username or password. This is generally a better security practice, but is  
more cumbersome, as it requires users to acquire these cookies themselves and then input them. 

To view steps on how to obtain your cookie and login securely, simply call `/login` with no arguments, or see the [annotated image here](./data/Cooke_Instructions.png)