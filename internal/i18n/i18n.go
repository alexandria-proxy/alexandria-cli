package i18n

type Strings struct {
	ChooseLanguage string
	Hint           string
	Connect        string
	Disconnect     string

	ServersTitle string
	SearchHint   string
	Autoupdate   string
	Expires      string
	NoSubs       string
	AddSubHint   string
}

var table = map[string]Strings{
	"en": {
		ChooseLanguage: "Choose language",
		Hint:           "↑/↓ move • enter select • q quit",
		Connect:        "Connect",
		Disconnect:     "Disconnect",
		ServersTitle:   "Servers",
		SearchHint:     "Type here to search",
		Autoupdate:     "autoupdate",
		Expires:        "Expires",
		NoSubs:         "No subscriptions found",
		AddSubHint:     "Press «a» to add",
	},
	"ru": {
		ChooseLanguage: "Выберите язык",
		Hint:           "↑/↓ выбор • enter выбрать • q выход",
		Connect:        "Подключиться",
		Disconnect:     "Отключиться",

		ServersTitle: "Сервера",
		SearchHint:   "Введите для поиска",
		Autoupdate:   "автообновление",
		Expires:      "Истекает",
		NoSubs:       "Подписки не найдены",
		AddSubHint:   "Нажмите «a» чтобы добавить",
	},
	"fa": {
		ChooseLanguage: "زبان را انتخاب کنید",
		Hint:           "↑/↓ حرکت • enter انتخاب • q خروج",
		Connect:        "اتصال",
		Disconnect:     "قطع اتصال",

		ServersTitle: "سرورها",
		SearchHint:   "برای جستجو تایپ کنید",
		Autoupdate:   "بروزرسانی خودکار",
		Expires:      "انقضا",
		NoSubs:       "اشتراکی یافت نشد",
		AddSubHint:   "«a» را برای افزودن بزنید",
	},
}

func T(code string) Strings {
	if s, ok := table[code]; ok {
		return s
	}
	return table["en"]
}
