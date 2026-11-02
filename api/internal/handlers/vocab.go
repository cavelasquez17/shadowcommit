package handlers

// option is a selectable value plus the copy the UI shows for it.
// Labels live server-side so the seed data, the API and the UI never disagree.
type option struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Emoji string `json:"emoji,omitempty"`
	// Snark is the sarcastic subtitle. This is a feature, not a bug.
	Snark string `json:"snark,omitempty"`
	Color string `json:"color,omitempty"`
}

var seniorityOptions = []option{
	{Value: "intern", Label: "Intern", Emoji: "🍼", Snark: "Unpaid, unbothered, unemployable.", Color: "#7dd3fc"},
	{Value: "junior", Label: "Junior", Emoji: "🐣", Snark: "Cheaper than the API, for now.", Color: "#86efac"},
	{Value: "mid", Label: "Mid-level", Emoji: "🧗", Snark: "Peak productivity, peak expendability.", Color: "#fde047"},
	{Value: "senior", Label: "Senior", Emoji: "🧙", Snark: "Paid to say 'it depends'.", Color: "#fdba74"},
	{Value: "staff", Label: "Staff", Emoji: "📐", Snark: "Writes documents nobody reads.", Color: "#f9a8d4"},
	{Value: "principal", Label: "Principal", Emoji: "👑", Snark: "Has not opened an IDE since 2021.", Color: "#c4b5fd"},
	{Value: "lead", Label: "Tech Lead", Emoji: "🎯", Snark: "Absorbs blame, distributes standups.", Color: "#67e8f9"},
	{Value: "cto", Label: "CTO", Emoji: "🛥️", Snark: "The one who saw the demo.", Color: "#fca5a5"},
	{Value: "shadow", Label: "Shadow Dev", Emoji: "🥷", Snark: "Does two jobs, exists in zero org charts.", Color: "#a3a3a3"},
	{Value: "freelance_survivor", Label: "Freelance Survivor", Emoji: "🏴‍☠️", Snark: "No boss to fire you. Also no salary.", Color: "#fbbf24"},
}

var causeOptions = []option{
	{Value: "copilot_did_it_faster", Label: "Copilot did it faster", Emoji: "🤖",
		Snark: "In its defense, it did.", Color: "#22d3ee"},
	{Value: "ceo_saw_a_demo", Label: "The CEO saw a demo", Emoji: "📺",
		Snark: "Two minutes on LinkedIn. Forty jobs.", Color: "#f472b6"},
	{Value: "shadow_work_exposed", Label: "Shadow work exposed", Emoji: "🕵️",
		Snark: "You committed with your real name. Rookie mistake.", Color: "#a78bfa"},
	{Value: "replaced_by_a_prompt", Label: "Replaced by a prompt", Emoji: "📝",
		Snark: "Eleven words. That was your whole role.", Color: "#4ade80"},
	{Value: "replaced_by_an_agent_swarm", Label: "Replaced by an agent swarm", Emoji: "🐝",
		Snark: "They needed twelve agents to do your job. Take the win.", Color: "#facc15"},
	{Value: "synergistic_restructuring", Label: "Synergistic restructuring", Emoji: "📊",
		Snark: "HR called it a 'shape change'. You called it Tuesday.", Color: "#fb923c"},
	{Value: "left_before_they_could_fire_me", Label: "Left before they could fire me", Emoji: "🚪",
		Snark: "Technically undefeated.", Color: "#38bdf8"},
	{Value: "still_alive_somehow", Label: "Still alive, somehow", Emoji: "🫡",
		Snark: "Refresh your calendar. There is a meeting.", Color: "#e879f9"},
}

var reactionOptions = []option{
	{Value: "f", Label: "Press F", Emoji: "💀", Snark: "Respects paid.", Color: "#f87171"},
	{Value: "relate", Label: "Too real", Emoji: "🫠", Snark: "This happened to me on a Thursday.", Color: "#fbbf24"},
	{Value: "laugh", Label: "Laughing", Emoji: "😂", Snark: "Crying, actually.", Color: "#facc15"},
	{Value: "robot", Label: "The machine did this", Emoji: "🤖", Snark: "Beep boop, you're fired.", Color: "#22d3ee"},
	{Value: "copium", Label: "Copium", Emoji: "🫡", Snark: "It's fine. Everything is fine.", Color: "#4ade80"},
	{Value: "brutal", Label: "No notes", Emoji: "🔥", Snark: "Devastating. Perfect.", Color: "#fb923c"},
	{Value: "lawyer", Label: "You should sue", Emoji: "⚖️", Snark: "Not legal advice. Obviously.", Color: "#c084fc"},
}

// languageOptions are the languages the forum accepts posts in, and the UI locales we ship.
var languageOptions = []option{
	{Value: "en", Label: "English", Emoji: "🇬🇧"},
	{Value: "es", Label: "Español", Emoji: "🇪🇸"},
	{Value: "pt", Label: "Português", Emoji: "🇧🇷"},
	{Value: "fr", Label: "Français", Emoji: "🇫🇷"},
	{Value: "de", Label: "Deutsch", Emoji: "🇩🇪"},
	{Value: "it", Label: "Italiano", Emoji: "🇮🇹"},
	{Value: "ru", Label: "Русский", Emoji: "🇷🇺"},
	{Value: "hi", Label: "हिन्दी", Emoji: "🇮🇳"},
	{Value: "zh", Label: "中文", Emoji: "🇨🇳"},
	{Value: "ja", Label: "日本語", Emoji: "🇯🇵"},
	{Value: "ko", Label: "한국어", Emoji: "🇰🇷"},
	{Value: "ar", Label: "العربية", Emoji: "🇸🇦"},
	{Value: "tr", Label: "Türkçe", Emoji: "🇹🇷"},
	{Value: "pl", Label: "Polski", Emoji: "🇵🇱"},
	{Value: "uk", Label: "Українська", Emoji: "🇺🇦"},
	{Value: "nl", Label: "Nederlands", Emoji: "🇳🇱"},
	{Value: "id", Label: "Bahasa Indonesia", Emoji: "🇮🇩"},
	{Value: "vi", Label: "Tiếng Việt", Emoji: "🇻🇳"},
}

// countryOptions carries ISO 3166-1 alpha-2 codes. The UI derives the flag emoji
// from the code, so this list only needs the English name.
var countryOptions = []option{
	{Value: "AR", Label: "Argentina"}, {Value: "AU", Label: "Australia"},
	{Value: "AT", Label: "Austria"}, {Value: "BD", Label: "Bangladesh"},
	{Value: "BE", Label: "Belgium"}, {Value: "BO", Label: "Bolivia"},
	{Value: "BR", Label: "Brazil"}, {Value: "BG", Label: "Bulgaria"},
	{Value: "CA", Label: "Canada"}, {Value: "CL", Label: "Chile"},
	{Value: "CN", Label: "China"}, {Value: "CO", Label: "Colombia"},
	{Value: "CR", Label: "Costa Rica"}, {Value: "HR", Label: "Croatia"},
	{Value: "CZ", Label: "Czechia"}, {Value: "DK", Label: "Denmark"},
	{Value: "DO", Label: "Dominican Republic"}, {Value: "EC", Label: "Ecuador"},
	{Value: "EG", Label: "Egypt"}, {Value: "SV", Label: "El Salvador"},
	{Value: "EE", Label: "Estonia"}, {Value: "FI", Label: "Finland"},
	{Value: "FR", Label: "France"}, {Value: "GE", Label: "Georgia"},
	{Value: "DE", Label: "Germany"}, {Value: "GH", Label: "Ghana"},
	{Value: "GR", Label: "Greece"}, {Value: "GT", Label: "Guatemala"},
	{Value: "HN", Label: "Honduras"}, {Value: "HK", Label: "Hong Kong"},
	{Value: "HU", Label: "Hungary"}, {Value: "IS", Label: "Iceland"},
	{Value: "IN", Label: "India"}, {Value: "ID", Label: "Indonesia"},
	{Value: "IE", Label: "Ireland"}, {Value: "IL", Label: "Israel"},
	{Value: "IT", Label: "Italy"}, {Value: "JM", Label: "Jamaica"},
	{Value: "JP", Label: "Japan"}, {Value: "KE", Label: "Kenya"},
	{Value: "LV", Label: "Latvia"}, {Value: "LT", Label: "Lithuania"},
	{Value: "MY", Label: "Malaysia"}, {Value: "MX", Label: "Mexico"},
	{Value: "MA", Label: "Morocco"}, {Value: "NL", Label: "Netherlands"},
	{Value: "NZ", Label: "New Zealand"}, {Value: "NG", Label: "Nigeria"},
	{Value: "NO", Label: "Norway"}, {Value: "PK", Label: "Pakistan"},
	{Value: "PA", Label: "Panama"}, {Value: "PY", Label: "Paraguay"},
	{Value: "PE", Label: "Peru"}, {Value: "PH", Label: "Philippines"},
	{Value: "PL", Label: "Poland"}, {Value: "PT", Label: "Portugal"},
	{Value: "RO", Label: "Romania"}, {Value: "RU", Label: "Russia"},
	{Value: "SA", Label: "Saudi Arabia"}, {Value: "RS", Label: "Serbia"},
	{Value: "SG", Label: "Singapore"}, {Value: "SK", Label: "Slovakia"},
	{Value: "SI", Label: "Slovenia"}, {Value: "ZA", Label: "South Africa"},
	{Value: "KR", Label: "South Korea"}, {Value: "ES", Label: "Spain"},
	{Value: "LK", Label: "Sri Lanka"}, {Value: "SE", Label: "Sweden"},
	{Value: "CH", Label: "Switzerland"}, {Value: "TW", Label: "Taiwan"},
	{Value: "TH", Label: "Thailand"}, {Value: "TN", Label: "Tunisia"},
	{Value: "TR", Label: "Türkiye"}, {Value: "UA", Label: "Ukraine"},
	{Value: "AE", Label: "United Arab Emirates"}, {Value: "GB", Label: "United Kingdom"},
	{Value: "US", Label: "United States"}, {Value: "UY", Label: "Uruguay"},
	{Value: "UZ", Label: "Uzbekistan"}, {Value: "VE", Label: "Venezuela"},
	{Value: "VN", Label: "Vietnam"},
}

func isValidOption(opts []option, v string) bool {
	for _, o := range opts {
		if o.Value == v {
			return true
		}
	}
	return false
}
