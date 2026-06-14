# Main Platform Bot UX/UI Specification

## Design Principles

### Reply Keyboard Usage

Use Reply Keyboard only for top-level navigation.

Reason:

* Always visible
* Easy access
* Better UX
* Less confusion

---

### Inline Keyboard Usage

Use Inline Keyboard for:

* Actions
* Details
* Editing
* Pagination
* Confirmations

Reason:

* Cleaner UI
* Context aware
* Easier localization

---

# Main Menu (Reply Keyboard)

Row 1

💰 Wallet
🤖 My Services

Row 2

🏘 Communities
📢 Advertisements

Row 3

📊 Earnings
💎 Plans

Row 4

🔔 Notifications
⚙️ Settings

---

# Wallet

## Wallet Home

Show:

Balance

Available Balance

Pending Balance

Today's Earnings

Buttons (Inline)

➕ Deposit

➖ Withdraw

📜 Transactions

🎁 Rewards

🔄 Transfer

---

## Deposit

Inline

Available Methods

💳 Crypto

🏦 Bank

🎁 Gift Code

---

## Withdraw

Inline

Available Methods

💳 Crypto

🏦 Bank

---

## Transactions

Inline Pagination

⬅️ Previous

➡️ Next

🔍 Details

---

# My Services

## Home

Inline

➕ Create Service

📋 My Services

📊 Statistics

---

## Create Service

Step 1

Choose Service Type

Inline

🌐 VPN

📤 Uploader

🔒 Membership Lock

📦 Future Services

---

Step 2

Choose Plan

Inline

Free

Starter

Pro

Enterprise

---

Step 3

Confirmation

Inline

✅ Create

❌ Cancel

---

## Service Details

Inline

📊 Statistics

⚙️ Settings

💰 Revenue

📈 Usage

🔄 Restart

⏸ Suspend

🗑 Delete

---

# Communities

## Communities Home

Inline

➕ Register Community

📋 My Communities

📊 Statistics

💰 Revenue

🏆 Leaderboard

---

## Register Community

Step 1

Select Type

Inline

👥 Group

📢 Channel

---

Step 2

Send Invite Link

Message Input

---

Step 3

Verification

Inline

✅ Verify

🔄 Retry

---

## Community Details

Inline

📊 Statistics

👥 Members

💰 Revenue

📈 Growth

🔗 Invite Links

⚙️ Settings

---

# Advertisements

## Advertisements Home

Inline

➕ Create Campaign

📋 Campaigns

📊 Statistics

💰 Budget

---

## Create Campaign

Step 1

Campaign Name

Message Input

---

Step 2

Budget

Message Input

---

Step 3

Duration

Inline

1 Day

3 Days

7 Days

30 Days

---

Step 4

Confirmation

Inline

✅ Launch

❌ Cancel

---

## Campaign Details

Inline

📊 Statistics

👥 Participants

💰 Revenue

⚙️ Edit

⏸ Pause

▶ Resume

🗑 Stop

---

# Earnings

## Earnings Home

Inline

💰 Total Earnings

📅 Daily

📆 Monthly

🏘 Community Revenue

🤖 Service Revenue

---

## Revenue Details

Inline

📊 Chart

📜 History

💸 Withdraw

---

# Plans

## Plans Home

Inline

💎 Current Plan

📦 Upgrade

📊 Compare Plans

📜 History

---

## Plan Details

Inline

Bots Limit

Resources

Features

Upgrade Button

---

# Notifications

## Home

Inline

💰 Financial

📢 Campaigns

🏘 Communities

⚙️ System

---

## Notification Settings

Inline Toggle Buttons

🔔 On

🔕 Off

---

# Settings

## Settings Home

Inline

🌍 Language

🔐 Security

💳 Payment Methods

🎨 Appearance

📞 Support

---

# Language

Inline

🇺🇸 English

🇮🇷 فارسی

🇷🇺 Русский

🇹🇷 Türkçe

🇦🇪 العربية

More...

---

# Security

Inline

🔑 Change Password

📱 Devices

🛡 Sessions

🔒 2FA

---

# Support

Inline

📚 Help Center

💬 Contact Support

🐞 Report Issue

---

# Admin Main Menu

Reply Keyboard

Row 1

👥 Users
🤖 Services

Row 2

🏘 Communities
📢 Campaigns

Row 3

💰 Finance
🚨 Fraud

Row 4

📈 Statistics
⚙️ System

---

# Admin Users

Inline

🔍 Search

📋 List

🚫 Suspended

⭐ Top Users

---

# Admin Services

Inline

🔍 Search

📋 List

📊 Statistics

🛑 Suspended

---

# Admin Communities

Inline

🔍 Search

📋 List

📈 Statistics

🚨 Suspicious

---

# Admin Campaigns

Inline

🔍 Search

📋 Active

📋 Finished

📊 Statistics

---

# Admin Finance

Inline

💰 Revenue

📤 Withdrawals

📊 Reports

💳 Wallets

---

# Admin Fraud

Inline

🚩 Events

👤 User Scores

🏘 Community Scores

🔍 Investigations

---

# Admin System

Inline

📦 Plans

🖥 Servers

🔌 Membership Bots

📡 NATS

🗄 Database

📈 Metrics

---

# Global UX Rules

1. Main navigation always uses Reply Keyboard.
2. Internal actions always use Inline Keyboard.
3. Never show more than 6 inline buttons in one section.
4. Every page must contain Back button.
5. Dangerous actions require confirmation.
6. Lists must support pagination.
7. All texts must be translation-key based.
8. Every button must have permission checks.
9. Admin and User flows must remain completely separated.
10. Every action should be executable within 2-3 clicks maximum.
11. All menus must support future Server Driven UI migration.
12. No hardcoded text inside handlers.
13. Button IDs must be stable and versioned.
14. Mobile-first UX only.
15. Statistics and revenue must always be separate screens.
