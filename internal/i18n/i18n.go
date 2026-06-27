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
	Used         string
	Of           string
	NoSubs       string
	AddSubHint   string

	AddSubTitle      string
	EditSubTitle     string
	FieldType        string
	FieldName        string
	FieldURL         string
	AddBtn           string
	TypeSubscription string
	TypeConfig       string
	TypeJSON         string
	Fetching         string

	EditServerTitle string
	EditHint        string
	SaveBtn         string

	ActionUpdate   string
	ActionTestPing string
	ActionPin      string
	ActionUnpin    string
	ActionCopyURL  string
	ActionEdit     string
	ActionRemove   string
	Updating       string
	Pinging        string
	Copied         string
	Dead           string
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
		Expires:        "expires",
		Used:           "used",
		Of:             "of",
		NoSubs:         "No subscriptions found",
		AddSubHint:     "Press «ctrl + a» to add",

		AddSubTitle:      "Add subscription",
		EditSubTitle:     "Edit subscription",
		FieldType:        "Type",
		FieldName:        "Subscription name",
		FieldURL:         "Subscription URL",
		AddBtn:           "Add",
		TypeSubscription: "Subscription",
		TypeConfig:       "Configuration",
		TypeJSON:         "JSON",
		Fetching:         "Fetching…",

		EditServerTitle: "Edit server configuration",
		EditHint:        "A wrong config will break the connection. Edit only if you know what you're doing.",
		SaveBtn:         "Save",

		ActionUpdate:   "update",
		ActionTestPing: "ping",
		ActionPin:      "pin",
		ActionUnpin:    "unpin",
		ActionCopyURL:  "copy url",
		ActionEdit:     "edit",
		ActionRemove:   "remove",
		Updating:       "updating…",
		Pinging:        "pinging…",
		Copied:         "copied!",
		Dead:           "dead",
	},
	"ru": {
		ChooseLanguage: "Выберите язык",
		Hint:           "↑/↓ выбор • enter выбрать • q выход",
		Connect:        "Подключиться",
		Disconnect:     "Отключиться",

		ServersTitle: "Серверы",
		SearchHint:   "Введите для поиска",
		Autoupdate:   "автообновление",
		Expires:      "истекает",
		Used:         "использовано",
		Of:           "из",
		NoSubs:       "Подписки не найдены",
		AddSubHint:   "Нажмите «ctrl + a» чтобы добавить",

		AddSubTitle:      "Добавление подписки",
		EditSubTitle:     "Редактирование подписки",
		FieldType:        "Тип",
		FieldName:        "Имя подписки",
		FieldURL:         "URL Подписки",
		AddBtn:           "Добавить",
		TypeSubscription: "Подписка",
		TypeConfig:       "Конфигурация",
		TypeJSON:         "JSON",
		Fetching:         "Загрузка…",

		EditServerTitle: "Редактирование конфигурации сервера",
		EditHint:        "Неверный конфиг порвёт соединение. Редактируйте только если знаете что делаете.",
		SaveBtn:         "Сохранить",

		ActionUpdate:   "обновить",
		ActionTestPing: "пинг",
		ActionPin:      "закрепить",
		ActionUnpin:    "открепить",
		ActionCopyURL:  "копировать url",
		ActionEdit:     "изменить",
		ActionRemove:   "удалить",
		Updating:       "обновление…",
		Pinging:        "проверка пинга…",
		Copied:         "скопировано!",
		Dead:           "dead",
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
		Used:         "استفاده شده",
		Of:           "از",
		NoSubs:       "اشتراکی یافت نشد",
		AddSubHint:   "«ctrl + a» را برای افزودن بزنید",

		AddSubTitle:      "افزودن اشتراک",
		EditSubTitle:     "ویرایش اشتراک",
		FieldType:        "نوع",
		FieldName:        "نام اشتراک",
		FieldURL:         "آدرس اشتراک",
		AddBtn:           "افزودن",
		TypeSubscription: "اشتراک",
		TypeConfig:       "پیکربندی",
		TypeJSON:         "JSON",
		Fetching:         "در حال دریافت…",

		EditServerTitle: "ویرایش پیکربندی سرور",
		EditHint:        "پیکربندی نادرست اتصال را قطع می‌کند. فقط اگر می‌دانید چه می‌کنید ویرایش کنید.",
		SaveBtn:         "ذخیره",

		ActionUpdate:   "بروزرسانی",
		ActionTestPing: "پینگ",
		ActionPin:      "سنجاق",
		ActionUnpin:    "برداشتن سنجاق",
		ActionCopyURL:  "کپی آدرس",
		ActionEdit:     "ویرایش",
		ActionRemove:   "حذف",
		Updating:       "در حال بروزرسانی…",
		Pinging:        "در حال پینگ…",
		Copied:         "کپی شد!",
		Dead:           "dead",
	},
}

func T(code string) Strings {
	if s, ok := table[code]; ok {
		return s
	}
	return table["en"]
}
