package db

// This file is only DATA: the sample catalog. The logic that inserts it is in seed.go.
//
// A "game" can be sold in several variants (PC Steam key, PS5 key, ...). Every variant
// becomes its own row in the products table, just like on Eneba, where
// "ELDEN RING (PC) Steam Key GLOBAL" and "ELDEN RING (PS5) PSN Key EUROPE" are
// separate products. 31 games in 68 variants + 12 gift cards/subscriptions = 80 products.

// variant = one platform/device/region combination of a game.
type variant struct {
	Platform string // filter value: Steam, PlayStation, Xbox, Nintendo
	Device   string // shown in the title: PC, PS5, PS4, Xbox Series X|S, Xbox One, Switch
	Region   string // Global, Europe, United States
}

type game struct {
	Name        string
	Genre       string
	Description string
	ReleaseDate string // YYYY-MM-DD
	MSRPCents   int    // official price in euro cents
	Popularity  int    // base sales count, used for "best sellers"
	Variants    []variant
}

// Short helpers so the list below stays readable.
var (
	pcGlobal   = variant{"Steam", "PC", "Global"}
	ps5Europe  = variant{"PlayStation", "PS5", "Europe"}
	ps4Europe  = variant{"PlayStation", "PS4", "Europe"}
	xsxGlobal  = variant{"Xbox", "Xbox Series X|S", "Global"}
	xoneGlobal = variant{"Xbox", "Xbox One", "Global"}
	swEurope   = variant{"Nintendo", "Switch", "Europe"}
	swUS       = variant{"Nintendo", "Switch", "United States"}
)

var games = []game{
	{"Elden Ring", "RPG", "Explore the Lands Between in FromSoftware's open-world action RPG, full of brutal bosses and hidden secrets.", "2022-02-25", 5999, 9200, []variant{pcGlobal, ps5Europe, xsxGlobal}},
	{"Cyberpunk 2077", "RPG", "Become V, a mercenary in Night City, and shape your story in this open-world RPG set in a dark future.", "2020-12-10", 5999, 8100, []variant{pcGlobal, ps5Europe, xsxGlobal}},
	{"The Witcher 3: Wild Hunt Complete Edition", "RPG", "Hunt monsters as Geralt of Rivia in a huge open world. Includes both story expansions.", "2015-05-19", 4999, 7400, []variant{pcGlobal, ps5Europe, swEurope}},
	{"Red Dead Redemption 2", "Action", "Ride with Arthur Morgan and the Van der Linde gang in an epic tale of life in America's unforgiving heartland.", "2018-10-26", 5999, 8800, []variant{pcGlobal, ps4Europe, xoneGlobal}},
	{"Grand Theft Auto V", "Action", "Three criminals, one city. Story mode plus access to the ever-growing world of GTA Online.", "2013-09-17", 2999, 9900, []variant{pcGlobal, ps5Europe, xsxGlobal}},
	{"Baldur's Gate 3", "RPG", "Gather your party and return to the Forgotten Realms in a story of fellowship, betrayal and survival.", "2023-08-03", 5999, 8600, []variant{pcGlobal, ps5Europe}},
	{"Hogwarts Legacy", "RPG", "Live the unwritten as a student at Hogwarts in the 1800s and explore an open world of magic.", "2023-02-10", 5999, 7900, []variant{pcGlobal, ps5Europe, xsxGlobal, swEurope}},
	{"EA SPORTS FC 25", "Sports", "The world's game with over 19,000 players, 700 teams and Rush, a new 5v5 mode.", "2024-09-27", 6999, 8300, []variant{pcGlobal, ps5Europe, xsxGlobal}},
	{"Minecraft", "Sandbox", "Build anything you can imagine, survive the night and explore infinite worlds alone or with friends.", "2011-11-18", 2999, 9500, []variant{xoneGlobal, swEurope, ps4Europe}},
	{"Call of Duty: Black Ops 6", "Shooter", "A spy thriller set in the early 90s, with a cinematic campaign, multiplayer and Zombies.", "2024-10-25", 6999, 8000, []variant{pcGlobal, ps5Europe, xsxGlobal}},
	{"Helldivers 2", "Shooter", "Join the Helldivers and spread managed democracy across the galaxy in a chaotic co-op shooter.", "2024-02-08", 3999, 6900, []variant{pcGlobal, ps5Europe}},
	{"Forza Horizon 5", "Racing", "Lead breathtaking expeditions across the vibrant open world of Mexico with hundreds of cars.", "2021-11-09", 5999, 6100, []variant{pcGlobal, xsxGlobal}},
	{"Starfield", "RPG", "Bethesda's first new universe in 25 years. Create a character and explore the stars with total freedom.", "2023-09-06", 6999, 5200, []variant{pcGlobal, xsxGlobal}},
	{"The Legend of Zelda: Tears of the Kingdom", "Adventure", "Explore the land and skies of Hyrule as Link in this sequel to Breath of the Wild.", "2023-05-12", 6999, 7100, []variant{swEurope, swUS}},
	{"Mario Kart 8 Deluxe", "Racing", "Race and battle your friends on 48 tracks with Mario and friends, at home or on the go.", "2017-04-28", 5999, 7600, []variant{swEurope, swUS}},
	{"Super Mario Bros. Wonder", "Platformer", "Classic side-scrolling Mario fun, with Wonder Flowers that change everything you know.", "2023-10-20", 5999, 5800, []variant{swEurope}},
	{"Animal Crossing: New Horizons", "Simulation", "Escape to a deserted island and create your own paradise at your own pace.", "2020-03-20", 5999, 6400, []variant{swEurope}},
	{"Pokémon Scarlet", "RPG", "Explore the open-world Paldea region, catch Pokémon and become a champion.", "2022-11-18", 5999, 6000, []variant{swEurope}},
	{"God of War Ragnarök", "Action", "Kratos and Atreus journey through the Nine Realms as Fimbulwinter comes to an end.", "2022-11-09", 5999, 6700, []variant{pcGlobal, ps5Europe}},
	{"Marvel's Spider-Man 2", "Action", "Peter Parker and Miles Morales face Venom in an expanded Marvel's New York.", "2023-10-20", 6999, 6300, []variant{pcGlobal, ps5Europe}},
	{"Ghost of Tsushima Director's Cut", "Action", "A samurai's journey to free Tsushima Island from the Mongol invasion. Includes the Iki Island expansion.", "2021-08-20", 5999, 5600, []variant{pcGlobal, ps5Europe}},
	{"Stardew Valley", "Simulation", "Inherit your grandfather's old farm and build the life you've always wanted in the countryside.", "2016-02-26", 1399, 7300, []variant{pcGlobal, swEurope}},
	{"Hollow Knight", "Platformer", "Descend into a vast ruined kingdom of insects and heroes in this hand-drawn action adventure.", "2017-02-24", 1499, 5400, []variant{pcGlobal, swEurope}},
	{"Hades", "Roguelike", "Defy the god of the dead as you hack and slash out of the Underworld in this rogue-like dungeon crawler.", "2020-09-17", 2499, 5100, []variant{pcGlobal, swEurope}},
	{"Sea of Thieves", "Adventure", "Set sail with friends, hunt treasure and battle other pirate crews in a shared open world.", "2018-03-20", 3999, 4900, []variant{pcGlobal, xsxGlobal}},
	{"Resident Evil 4", "Horror", "A remake of the survival horror classic. Rescue the president's daughter from a mysterious cult.", "2023-03-24", 3999, 5700, []variant{pcGlobal, ps5Europe}},
	{"Diablo IV", "RPG", "Fight the forces of Hell in a dark open world, alone or with friends.", "2023-06-06", 6999, 4800, []variant{ps5Europe, xsxGlobal}},
	{"It Takes Two", "Adventure", "A co-op adventure about a couple on the brink of divorce, magically turned into dolls.", "2021-03-26", 3999, 5300, []variant{pcGlobal, ps5Europe}},
	{"Palworld", "Survival", "Catch creatures called Pals, build a base, and fight, farm and craft together in a survival world.", "2024-01-19", 2899, 6200, []variant{pcGlobal}},
	{"Black Myth: Wukong", "Action", "An action RPG rooted in Chinese mythology. Become the Destined One and uncover the truth behind a legend.", "2024-08-20", 5999, 6600, []variant{pcGlobal, ps5Europe}},
	{"Final Fantasy VII Rebirth", "RPG", "Cloud and his friends leave Midgar behind and set out on a journey across a vast world.", "2024-02-29", 7999, 4600, []variant{pcGlobal, ps5Europe}},
}

// giftCard = a product that is not a game: store credit or a subscription.
type giftCard struct {
	Name       string // full card name, e.g. "Steam Gift Card 20 EUR"
	Genre      string // "Gift Card" or "Subscription"
	Platform   string
	Store      string // shown in the title: "Steam", "PSN", ...
	Region     string
	FaceCents  int // the card's value = its MSRP
	Popularity int
}

var giftCards = []giftCard{
	{"Steam Gift Card 10 EUR", "Gift Card", "Steam", "Steam", "Europe", 1000, 6800},
	{"Steam Gift Card 20 EUR", "Gift Card", "Steam", "Steam", "Europe", 2000, 7700},
	{"Steam Gift Card 50 EUR", "Gift Card", "Steam", "Steam", "Europe", 5000, 5900},
	{"Steam Gift Card 20 USD", "Gift Card", "Steam", "Steam", "United States", 1850, 4400},
	{"PlayStation Store Gift Card 20 EUR", "Gift Card", "PlayStation", "PSN", "Europe", 2000, 6500},
	{"PlayStation Store Gift Card 50 EUR", "Gift Card", "PlayStation", "PSN", "Europe", 5000, 5000},
	{"PlayStation Plus Essential 12 Months", "Subscription", "PlayStation", "PSN", "Europe", 7199, 5500},
	{"Xbox Gift Card 25 EUR", "Gift Card", "Xbox", "Xbox Live", "Europe", 2500, 4700},
	{"Xbox Gift Card 50 EUR", "Gift Card", "Xbox", "Xbox Live", "Europe", 5000, 3900},
	{"Xbox Game Pass Ultimate 1 Month", "Subscription", "Xbox", "Xbox Live", "Global", 1799, 7000},
	{"Nintendo eShop Card 25 EUR", "Gift Card", "Nintendo", "Nintendo eShop", "Europe", 2500, 4300},
	{"Nintendo eShop Card 50 EUR", "Gift Card", "Nintendo", "Nintendo eShop", "Europe", 5000, 3600},
}

// Fictional marketplace sellers.
type seller struct {
	Name          string
	RatingPercent int
	SalesCount    int
}

var sellers = []seller{
	{"PixelVault", 99, 184000},
	{"KeyNest", 98, 152300},
	{"GameDrop", 97, 98700},
	{"LootLane", 96, 76400},
	{"NovaKeys", 99, 64100},
	{"QuestMarket", 94, 41800},
	{"CodeHaven", 92, 23500},
	{"Starlight Games", 95, 12900},
}

// Homepage blocks: the frontend renders these in order, and each block
// fetches its own content from Endpoint. Nothing about the homepage is hardcoded in React.
type homepageBlock struct {
	Type     string
	Title    string
	Endpoint string
}

var homepageBlocks = []homepageBlock{
	{"banner", "Featured", "/api/home/banners"},
	{"product_row", "Best sellers", "/api/home/best-sellers"},
	{"product_row", "Top deals", "/api/home/deals"},
}

// Banner slides. Slug points to the product the banner links to.
type banner struct {
	Title    string
	Subtitle string
	Slug     string
}

var banners = []banner{
	{"ELDEN RING", "Rise, Tarnished. Save up to 45% on the PC key.", "elden-ring-pc-steam-global"},
	{"Baldur's Gate 3", "The RPG of the year, now at a better price.", "baldur-s-gate-3-pc-steam-global"},
	{"Xbox Game Pass Ultimate", "Hundreds of games for one monthly price.", "xbox-game-pass-ultimate-1-month-xbox-live-global"},
}
