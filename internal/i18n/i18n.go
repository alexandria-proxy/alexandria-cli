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

	AddSubTitle      string
	FieldType        string
	FieldName        string
	FieldURL         string
	AddBtn           string
	TypeSubscription string
	TypeConfig       string
	TypeJSON         string
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
		AddSubHint:     "Press «ctrl + a» to add",

		AddSubTitle:      "Add subscription",
		FieldType:        "Type",
		FieldName:        "Subscription name",
		FieldURL:         "Subscription URL",
		AddBtn:           "Add",
		TypeSubscription: "Subscription",
		TypeConfig:       "Configuration",
		TypeJSON:         "JSON",
	},
	"ru": {
		ChooseLanguage: "Выберите язык",
		Hint:           "↑/↓ выбор • enter выбрать • q выход",
		Connect:        "Подключиться",
		Disconnect:     "Отключиться",

		ServersTitle: "Серверы",
		SearchHint:   "Введите для поиска",
		Autoupdate:   "автообновление",
		Expires:      "Истекает",
		NoSubs:       "Подписки не найдены",
		AddSubHint:   "Нажмите «ctrl + a» чтобы добавить",

		AddSubTitle:      "Добавление подписки",
		FieldType:        "Тип",
		FieldName:        "Имя подписки",
		FieldURL:         "URL Подписки",
		AddBtn:           "Добавить",
		TypeSubscription: "Подписка",
		TypeConfig:       "Конфигурация",
		TypeJSON:         "JSON",
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
		AddSubHint:   "«ctrl + a» را برای افزودن بزنید",

		AddSubTitle:      "افزودن اشتراک",
		FieldType:        "نوع",
		FieldName:        "نام اشتراک",
		FieldURL:         "آدرس اشتراک",
		AddBtn:           "افزودن",
		TypeSubscription: "اشتراک",
		TypeConfig:       "پیکربندی",
		TypeJSON:         "JSON",
	},
}

func T(code string) Strings {
	if s, ok := table[code]; ok {
		return s
	}
	return table["en"]
}
