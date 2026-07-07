package i18n

type Strings struct {
	ChooseLanguage string
	Hint           string
	Connect        string
	Connecting     string
	Disconnect     string

	ServersTitle string
	SearchHint   string
	Autoupdate   string
	Expires      string
	Updated      string
	Used         string
	Of           string
	NoteLabel    string
	NoSubs       string
	AddSubHint   string

	AddSubTitle      string
	EditSubTitle     string
	InfoTitle        string
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
	ActionInfo     string
	ActionRemove   string
	Updating       string
	Pinging        string
	Copied         string
	Dead           string
	NotFound       string
}

var table = map[string]Strings{
	"en": {
		ChooseLanguage: "Choose language",
		Hint:           "↑/↓ move • enter select • q quit",
		Connect:        "Connect",
		Connecting:     "Connecting…",
		Disconnect:     "Disconnect",
		ServersTitle:   "Servers",
		SearchHint:     "Type here to search",
		Autoupdate:     "autoupdate",
		Expires:        "expires",
		Updated:        "updated",
		Used:           "used",
		Of:             "of",
		NoteLabel:      "note",
		NoSubs:         "No subscriptions found",
		AddSubHint:     "Press «ctrl + a» to add",

		AddSubTitle:      "Add subscription",
		EditSubTitle:     "Edit subscription",
		InfoTitle:        "Subscription info",
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
		ActionInfo:     "info",
		ActionRemove:   "remove",
		Updating:       "updating…",
		Pinging:        "pinging…",
		Copied:         "copied!",
		Dead:           "dead",
		NotFound:       "not found",
	},
	"ru": {
		ChooseLanguage: "Выберите язык",
		Hint:           "↑/↓ выбор • enter выбрать • q выход",
		Connect:        "Подключиться",
		Connecting:     "Подключение…",
		Disconnect:     "Отключиться",

		ServersTitle: "Серверы",
		SearchHint:   "Введите для поиска",
		Autoupdate:   "автообновление",
		Expires:      "истекает",
		Updated:      "обновлено",
		Used:         "использовано",
		Of:           "из",
		NoteLabel:    "заметка",
		NoSubs:       "Подписки не найдены",
		AddSubHint:   "Нажмите «ctrl + a» чтобы добавить",

		AddSubTitle:      "Добавление подписки",
		EditSubTitle:     "Редактирование подписки",
		InfoTitle:        "Информация о подписке",
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
		ActionInfo:     "инфо",
		ActionRemove:   "удалить",
		Updating:       "обновление…",
		Pinging:        "проверка пинга…",
		Copied:         "скопировано!",
		Dead:           "dead",
		NotFound:       "не найдено",
	},
	"fa": {
		ChooseLanguage: "زبان را انتخاب کنید",
		Hint:           "↑/↓ حرکت • enter انتخاب • q خروج",
		Connect:        "اتصال",
		Connecting:     "در حال اتصال…",
		Disconnect:     "قطع اتصال",

		ServersTitle: "سرورها",
		SearchHint:   "برای جستجو تایپ کنید",
		Autoupdate:   "بروزرسانی خودکار",
		Expires:      "انقضا",
		Updated:      "بروزرسانی شده",
		Used:         "استفاده شده",
		Of:           "از",
		NoteLabel:    "یادداشت",
		NoSubs:       "اشتراکی یافت نشد",
		AddSubHint:   "«ctrl + a» را برای افزودن بزنید",

		AddSubTitle:      "افزودن اشتراک",
		EditSubTitle:     "ویرایش اشتراک",
		InfoTitle:        "اطلاعات اشتراک",
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
		ActionInfo:     "اطلاعات",
		ActionRemove:   "حذف",
		Updating:       "در حال بروزرسانی…",
		Pinging:        "در حال پینگ…",
		Copied:         "کپی شد!",
		Dead:           "dead",
		NotFound:       "یافت نشد",
	},
}

func T(code string) Strings {
	if s, ok := table[code]; ok {
		return s
	}
	return table["en"]
}
