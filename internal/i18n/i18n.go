package i18n

type Strings struct {
	ChooseLanguage string
	Hint           string
	Connect        string
	Disconnect     string
}

var table = map[string]Strings{
	"en": {
		ChooseLanguage: "Choose language",
		Hint:           "↑/↓ move • enter select • q quit",
		Connect:        "Connect",
		Disconnect:     "Disconnect",
	},
	"ru": {
		ChooseLanguage: "Выберите язык",
		Hint:           "↑/↓ выбор • enter выбрать • q выход",
		Connect:        "Подключиться",
		Disconnect:     "Отключиться",
	},
	"fa": {
		ChooseLanguage: "زبان را انتخاب کنید",
		Hint:           "↑/↓ حرکت • enter انتخاب • q خروج",
		Connect:        "اتصال",
		Disconnect:     "قطع اتصال",
	},
}

func T(code string) Strings {
	if s, ok := table[code]; ok {
		return s
	}
	return table["en"]
}
