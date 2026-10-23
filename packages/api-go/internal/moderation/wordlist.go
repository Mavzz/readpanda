package moderation

// blockedWords is the whole list (report-block R8.5): add or remove a word
// here and nothing else changes. Entries match as whole words, ignoring
// case, accents and look-alike characters; an entry of several words matches
// those words in a row. Plurals and spelling variants are separate entries.
//
// Slurs and explicit sexual terms only. Words that are also ordinary words
// stay off it ("dick" is in Moby Dick, "cock" is a rooster, "pussy" a cat):
// this is the first line, and reports catch the rest. The same goes for
// other languages: "randi" is also an English name, "lund" a Swedish word
// and a surname, "chhakka" a six in cricket, "sunni" a branch of Islam, so
// they stay off. So do words romanized like an ordinary one ("choot" is
// also how "chhoot", a discount, gets typed) and places and people
// ("katwa", "sulla", "lauda").
//
// Indian languages are listed romanized, the way most readers type them,
// and in their own script where we're sure of the spelling. A native
// speaker should check the script entries before each one is relied on.
var blockedWords = []string{
	// Racial and ethnic slurs
	"nigger", "niggers", "nigga", "niggas", "niggah",
	"jigaboo", "jigaboos",
	"porch monkey", "porch monkeys",
	"chink", "chinks", "chinky", "chinki", "chinkies",
	"gook", "gooks",
	"zipperhead", "zipperheads",
	"slanteye", "slanteyes",
	"spic", "spics",
	"beaner", "beaners",
	"wetback", "wetbacks",
	"dago", "dagos",
	"kike", "kikes",
	"raghead", "ragheads",
	"towelhead", "towelheads",
	"sand nigger", "sand niggers",
	"paki", "pakis",
	// UK and Ireland: Travellers and Roma
	"pikey", "pikeys",
	"gyppo", "gyppos",
	// Australia: Aboriginal people
	"boong", "boongs",

	// Slurs about sexuality, gender and disability
	"faggot", "faggots", "fag", "fags",
	"tranny", "trannies", "trannie",
	"shemale", "shemales",
	"retard", "retards", "retarded",
	"spaz",

	// Explicit sexual terms
	"cunt", "cunts",
	"blowjob", "blowjobs", "blow job", "blow jobs",
	"handjob", "handjobs", "hand job", "hand jobs",
	"rimjob", "rimjobs",
	"titjob", "titjobs",
	"cocksucker", "cocksuckers",
	"cumshot", "cumshots",
	"creampie", "gangbang", "bukkake",
	"jizz",
	"dildo", "dildos",
	"milf",

	// India: caste slurs
	"chamar", "chamaar", "chamaro",
	"bhangi", "bhangii",
	"chuhra", "chuhda",
	"dhed",
	"चमार", "भंगी", "चूहड़ा", "चूहडा",

	// India: religious and regional slurs
	"katua", "katuwa", "katue",
	"landya", "landye",
	"कटुआ",

	// Hindi, Urdu and Punjabi: sexual abuse
	"madarchod", "maderchod", "madarchood", "maadarchod", "madharchod", "motherchod",
	"behenchod", "bhenchod", "behanchod", "bhenchood", "banchod", "benchod", "pencho", "penchod",
	"bsdk",
	"bhosdike", "bhosadike", "bhosdi", "bhosda", "bhosdiwale", "bhosdiwala", "bhosadi", "bhosdk",
	"chutiya", "chutiye", "chutia", "chootiya", "chutiyapa", "chutiyo",
	"chudai", "chodu", "chudakkad",
	"lavda", "lawda", "lavde", "lawde", "lodu",
	"gaand", "gaandu", "gandu", "gaandmasti",
	"jhaatu", "jhaantu", "jhant", "jhaant",
	"randikhana", "randibaaz", "randwa",
	"phuddi", "phudi", "phuddu",
	"मादरचोद", "बहनचोद", "बहेनचोद", "भेनचोद", "भोसड़ीके", "भोसडीके", "भोसड़ी", "भोसडी",
	"चूतिया", "चुतिया", "चूत", "लौड़ा", "लौडा", "लवड़ा", "लौड़े", "गांडू", "गाण्डू", "गांड",
	"रंडी", "रण्डी", "झांटू", "चुदाई",

	// Marathi
	"zavadya", "zhavadya", "bhadkhau", "bhadkhav", "aizhavli",
	"झवाड्या", "भडखाऊ",

	// Bengali
	"khanki", "khankir", "bokachoda", "maagi",
	"খানকি", "বোকাচোদা", "মাগী",

	// Gujarati
	"bhosrina", "bhosadina",

	// Telugu
	"lanja", "lanjakodaka", "lanja kodaka", "dengutha", "denguta", "kojja", "modda",
	"లంజ", "లంజకొడక", "దెంగుతా", "కొజ్జా",

	// Tamil
	"pundai", "punda", "thevdiya", "thevidiya", "thevidiyaa", "oombu", "koothi", "ommala", "ommaala",
	"புண்டை", "தேவடியா", "கூதி",

	// Kannada
	"soole", "sule maga", "bolimaga", "boli maga", "soolemaga",
	"ಸೂಳೆ", "ಬೋಳಿಮಗ",

	// Malayalam
	"myre", "myru", "pooru", "poorimon", "poorimone", "thayoli", "thayolli", "kunna",
	"മൈര്", "പൂറ്", "തായോളി",
}
