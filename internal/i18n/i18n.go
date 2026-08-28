package i18n

type Strings struct {
	ChooseLanguage string
	Hint           string
	Connect        string
	Connecting     string
	Disconnect     string

	ServersTitle   string
	SettingsTitle  string
	LogsTitle      string
	AutostartLabel string
	AutostartHint  string
	SearchHint     string
	Autoupdate     string
	Expires        string
	Updated        string
	Used           string
	Of             string
	NoteLabel      string
	NoSubs         string
	AddSubHint     string

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

	SettingsAdvanced string
	SettingsSubs     string
	SettingsReset    string
	SettingsRoutes   string
	SetSearchHint    string

	SecGeneral       string
	SecConnection    string
	SecOther         string
	SecUpdate        string
	SecSending       string
	SecUserAgent     string
	SecServerList    string
	SecOtherSettings string
	SecCIDR          string
	SecLogging       string
	SecSources       string
	SecStorage       string

	LblLanguage     string
	LblPingProto    string
	LblPreferIP     string
	LblFrag         string
	LblFragPackets  string
	LblFragLength   string
	LblFragInterval string
	LblMux          string
	LblMuxConc      string
	LblLAN          string
	LblCurrentIP    string
	LblSocks        string
	NoteFrag        string
	NoteMux         string
	NoteLAN         string

	LblAutoUpdate  string
	LblUpdateEvery string
	LblTimeout     string
	NoteUpdate     string
	LblUpdateOpen  string
	LblPingOpen    string
	LblConnectOpen string
	NoteOnOpen     string
	LblNoDupes     string
	NoteNoDupes    string
	LblSendHWID    string
	NoteHWID       string
	NoteUserAgent  string

	LblLocalDNS    string
	LblJSONDNS     string
	LblResolveSrv  string
	LblSniffing    string
	LblSysProxy    string
	LblTUN         string
	LblTunProvider string
	LblTunMode     string
	LblTunConfig   string
	LblTunDNSOn    string
	LblTunDNS      string

	AddCIDR  string
	NoteCIDR string

	ResetUser      string
	NoteResetUser  string
	ResetPrefs     string
	NoteResetPrefs string
	ResetTun       string
	NoteResetTun   string
	ConfirmTitle   string
	ConfirmBtn     string
	CancelBtn      string

	OptAuto      string
	OptIPv4      string
	OptIPv6      string
	OptTCP       string
	OptICMP      string
	OptMixed     string
	OptSystem    string
	OptGvisor    string
	OptDefault   string
	OptCustom    string
	OptSingbox   string
	OptUnsorted  string
	OptSortPing  string
	OptSortAlpha string

	LblLogOn      string
	NoteLogOn     string
	LblLogDaemon  string
	LblLogXray    string
	LblLogSingbox string
	NoteLogSrc    string
	LblLogMax     string
	NoteLogMax    string
	OptNoLimit    string

	LogsFilterHint string
	LogsEmpty      string
	LogsFollow     string
	LogsPaused     string
	LogsLines      string
	LogsHint       string

	Seconds string
	Hours   string

	ConnLost         string
	ErrCrashLoop     string
	ErrAutostart     string
	ErrAutostartRoot string
	ErrBadCIDR       string
	ErrBadSize       string
	ErrBadPort       string
	ErrReset         string
	OkReset          string
}

var table = map[string]Strings{
	"en": {
		ChooseLanguage: "Choose language",
		Hint:           "↑/↓ move • enter select • q quit",
		Connect:        "Connect",
		Connecting:     "Connecting…",
		Disconnect:     "Disconnect",
		ServersTitle:   "Servers",
		SettingsTitle:  "Settings",
		LogsTitle:      "Logs",
		AutostartLabel: "Start on boot",
		AutostartHint:  "Launch alexandria with the system.",
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

		SettingsAdvanced: "Advanced settings",
		SettingsSubs:     "Subscriptions",
		SettingsReset:    "Reset",
		SettingsRoutes:   "Excluded routes",
		SetSearchHint:    "Search settings",

		SecGeneral:       "General",
		SecConnection:    "Connection",
		SecOther:         "Other",
		SecUpdate:        "Update",
		SecSending:       "Sending data",
		SecUserAgent:     "User agent",
		SecServerList:    "Server list",
		SecOtherSettings: "Other settings",
		SecCIDR:          "CIDR list",
		SecLogging:       "Logging",
		SecSources:       "Sources",
		SecStorage:       "Storage",

		LblLanguage:     "Language",
		LblPingProto:    "Ping protocol",
		LblPreferIP:     "Preferred IP type",
		LblFrag:         "Fragmentation",
		LblFragPackets:  "Packets",
		LblFragLength:   "Length",
		LblFragInterval: "Interval",
		LblMux:          "Mux",
		LblMuxConc:      "Concurrency",
		LblLAN:          "Allow connection from LAN",
		LblCurrentIP:    "Current IP",
		LblSocks:        "SOCKS5 proxy",
		NoteFrag:        "Splits the TLS handshake so DPI can't match it.",
		NoteMux:         "Several streams over one connection.",
		NoteLAN:         "Other devices can use this machine as a proxy.",

		LblAutoUpdate:  "Auto update",
		LblUpdateEvery: "Update interval (hours)",
		LblTimeout:     "Request timeout",
		NoteUpdate:     "How often subscriptions refresh while running.",
		LblUpdateOpen:  "Update on open",
		LblPingOpen:    "Ping on open",
		LblConnectOpen: "Connect on open",
		NoteOnOpen:     "Runs on every launch.",
		LblNoDupes:     "Ignore duplicates",
		NoteNoDupes:    "Hides servers already in the list.",
		LblSendHWID:    "Send HWID",
		NoteHWID:       "Sends X-Hwid, X-Device-Os and X-Device-Model. The id comes from the system and can't be changed.",
		NoteUserAgent:  "Clear the field to use the default.",

		LblLocalDNS:    "Use local DNS",
		LblJSONDNS:     "Use DNS from JSON",
		LblResolveSrv:  "Enable Resolve Server",
		LblSniffing:    "Enable sniffing",
		LblSysProxy:    "Set system proxy",
		LblTUN:         "TUN",
		LblTunProvider: "TUN provider",
		LblTunMode:     "TUN mode",
		LblTunConfig:   "TUN configuration",
		LblTunDNSOn:    "TUN DNS Enable",
		LblTunDNS:      "TUN DNS Address",

		AddCIDR:  "Add CIDR…",
		NoteCIDR: "These routes bypass the tunnel and go direct.",

		ResetUser:      "Reset user settings",
		NoteResetUser:  "Deletes all user data.",
		ResetPrefs:     "Reset settings",
		NoteResetPrefs: "Resets preferences, keeps subscriptions and servers.",
		ResetTun:       "Reset tunnel configuration",
		NoteResetTun:   "Clears every tun interface created earlier.",
		ConfirmTitle:   "Are you sure?",
		ConfirmBtn:     "Reset",
		CancelBtn:      "Cancel",

		OptAuto:      "auto",
		OptIPv4:      "ipv4",
		OptIPv6:      "ipv6",
		OptTCP:       "tcp/ip",
		OptICMP:      "icmp",
		OptMixed:     "mixed",
		OptSystem:    "system",
		OptGvisor:    "gvisor",
		OptDefault:   "default",
		OptCustom:    "custom",
		OptSingbox:   "sing-box",
		OptUnsorted:  "Unsorted",
		OptSortPing:  "Sort by Ping",
		OptSortAlpha: "Sort by Alphabet",

		LblLogOn:      "Enable logging",
		NoteLogOn:     "When off, nothing is recorded at all.",
		LblLogDaemon:  "Daemon events",
		LblLogXray:    "Xray",
		LblLogSingbox: "Sing-box",
		NoteLogSrc:    "Which sources end up in the log.",
		LblLogMax:     "Max log size",
		NoteLogMax:    "0 = no limit. Write it like 1 mb, 512 kb, 8 gb.",
		OptNoLimit:    "no limit",

		LogsFilterHint: "Filter",
		LogsEmpty:      "Nothing logged yet",
		LogsFollow:     "following",
		LogsPaused:     "paused",
		LogsLines:      "lines",
		LogsHint:       "ctrl + e to end • c clear",

		Seconds: "sec.",
		Hours:   "h",

		ConnLost:         "the connection dropped",
		ErrCrashLoop:     "the core keeps exiting — try another server",
		ErrAutostart:     "couldn't change autostart",
		ErrAutostartRoot: "autostart was set up system-wide — run alexandria with sudo to change it",
		ErrBadCIDR:       "that doesn't look like a CIDR or a domain",
		ErrBadSize:       "write a size like 1 mb, 512 kb or 8 gb",
		ErrBadPort:       "the port has to be between 1 and 65534",
		ErrReset:         "reset failed",
		OkReset:          "done",
	},
	"ru": {
		ChooseLanguage: "Выберите язык",
		Hint:           "↑/↓ выбор • enter выбрать • q выход",
		Connect:        "Подключиться",
		Connecting:     "Подключение…",
		Disconnect:     "Отключиться",

		ServersTitle:   "Серверы",
		SettingsTitle:  "Настройки",
		LogsTitle:      "Логи",
		AutostartLabel: "Автозапуск при загрузке",
		AutostartHint:  "Запускать alexandria вместе с системой.",
		SearchHint:     "Введите для поиска",
		Autoupdate:     "автообновление",
		Expires:        "истекает",
		Updated:        "обновлено",
		Used:           "использовано",
		Of:             "из",
		NoteLabel:      "заметка",
		NoSubs:         "Подписки не найдены",
		AddSubHint:     "Нажмите «ctrl + a» чтобы добавить",

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
		Dead:           "нет связи",
		NotFound:       "не найдено",

		SettingsAdvanced: "Дополнительные настройки",
		SettingsSubs:     "Подписки",
		SettingsReset:    "Сброс",
		SettingsRoutes:   "Исключённые маршруты",
		SetSearchHint:    "Поиск по настройкам",

		SecGeneral:       "Основное",
		SecConnection:    "Соединение",
		SecOther:         "Прочее",
		SecUpdate:        "Обновление",
		SecSending:       "Отправка данных",
		SecUserAgent:     "User agent",
		SecServerList:    "Список серверов",
		SecOtherSettings: "Прочие настройки",
		SecCIDR:          "Список CIDR",
		SecLogging:       "Логирование",
		SecSources:       "Источники",
		SecStorage:       "Хранение",

		LblLanguage:     "Язык",
		LblPingProto:    "Протокол пинга",
		LblPreferIP:     "Предпочитаемый тип IP",
		LblFrag:         "Фрагментация",
		LblFragPackets:  "Пакеты",
		LblFragLength:   "Длина",
		LblFragInterval: "Интервал",
		LblMux:          "Mux",
		LblMuxConc:      "Одновременных потоков",
		LblLAN:          "Разрешить подключения из локальной сети",
		LblCurrentIP:    "Текущий IP",
		LblSocks:        "SOCKS5 прокси",
		NoteFrag:        "Режет TLS-рукопожатие, чтобы DPI его не опознал.",
		NoteMux:         "Несколько потоков через одно соединение.",
		NoteLAN:         "Другие устройства смогут ходить через этот компьютер.",

		LblAutoUpdate:  "Автообновление",
		LblUpdateEvery: "Интервал обновления (часы)",
		LblTimeout:     "Таймаут запроса",
		NoteUpdate:     "Как часто обновляются подписки во время работы.",
		LblUpdateOpen:  "Обновлять при запуске",
		LblPingOpen:    "Пинговать при запуске",
		LblConnectOpen: "Подключаться при запуске",
		NoteOnOpen:     "Выполняется при каждом запуске.",
		LblNoDupes:     "Игнорировать дубликаты",
		NoteNoDupes:    "Скрывает серверы, которые уже есть в списке.",
		LblSendHWID:    "Отправлять HWID",
		NoteHWID:       "Отправляет X-Hwid, X-Device-Os и X-Device-Model. Идентификатор берётся из системы и не меняется.",
		NoteUserAgent:  "Очистите поле, чтобы вернуть значение по умолчанию.",

		LblLocalDNS:    "Использовать локальный DNS",
		LblJSONDNS:     "Использовать DNS из JSON",
		LblResolveSrv:  "Включить Resolve Server",
		LblSniffing:    "Включить sniffing",
		LblSysProxy:    "Системный прокси",
		LblTUN:         "TUN",
		LblTunProvider: "Движок TUN",
		LblTunMode:     "Режим TUN",
		LblTunConfig:   "Конфигурация TUN",
		LblTunDNSOn:    "Включить TUN DNS",
		LblTunDNS:      "Адрес TUN DNS",

		AddCIDR:  "Добавить CIDR…",
		NoteCIDR: "Эти маршруты идут мимо туннеля, напрямую.",

		ResetUser:      "Сбросить пользовательские данные",
		NoteResetUser:  "Удаляет все пользовательские данные.",
		ResetPrefs:     "Сбросить настройки",
		NoteResetPrefs: "Сбрасывает настройки, подписки и серверы остаются.",
		ResetTun:       "Сбросить конфигурацию туннеля",
		NoteResetTun:   "Удаляет все ранее созданные tun-интерфейсы.",
		ConfirmTitle:   "Вы уверены?",
		ConfirmBtn:     "Сбросить",
		CancelBtn:      "Отмена",

		OptAuto:      "авто",
		OptIPv4:      "ipv4",
		OptIPv6:      "ipv6",
		OptTCP:       "tcp/ip",
		OptICMP:      "icmp",
		OptMixed:     "смешанный",
		OptSystem:    "системный",
		OptGvisor:    "gvisor",
		OptDefault:   "по умолчанию",
		OptCustom:    "свой",
		OptSingbox:   "sing-box",
		OptUnsorted:  "Без сортировки",
		OptSortPing:  "По пингу",
		OptSortAlpha: "По алфавиту",

		LblLogOn:      "Вести логи",
		NoteLogOn:     "Когда выключено, не записывается ничего.",
		LblLogDaemon:  "События демона",
		LblLogXray:    "Xray",
		LblLogSingbox: "Sing-box",
		NoteLogSrc:    "Какие источники попадают в лог.",
		LblLogMax:     "Максимальный размер",
		NoteLogMax:    "0 = без предела. Пишите вида 1 mb, 512 kb, 8 gb.",
		OptNoLimit:    "без предела",

		LogsFilterHint: "Фильтр",
		LogsEmpty:      "Записей пока нет",
		LogsFollow:     "слежение",
		LogsPaused:     "пауза",
		LogsLines:      "строк",
		LogsHint:       "ctrl + e в конец • c очистить",

		Seconds: "сек.",
		Hours:   "ч",

		ConnLost:         "соединение разорвано",
		ErrCrashLoop:     "ядро постоянно падает — попробуйте другой сервер",
		ErrAutostart:     "не удалось изменить автозапуск",
		ErrAutostartRoot: "автозапуск поставлен на всю систему — запустите alexandria через sudo, чтобы его изменить",
		ErrBadCIDR:       "это не похоже на CIDR или домен",
		ErrBadSize:       "напишите размер вида 1 mb, 512 kb или 8 gb",
		ErrBadPort:       "порт должен быть от 1 до 65534",
		ErrReset:         "сбросить не удалось",
		OkReset:          "готово",
	},
	"fa": {
		ChooseLanguage: "زبان را انتخاب کنید",
		Hint:           "↑/↓ حرکت • enter انتخاب • q خروج",
		Connect:        "اتصال",
		Connecting:     "در حال اتصال…",
		Disconnect:     "قطع اتصال",

		ServersTitle:   "سرورها",
		SettingsTitle:  "تنظیمات",
		LogsTitle:      "گزارش‌ها",
		AutostartLabel: "اجرا هنگام روشن شدن",
		AutostartHint:  "اجرای alexandria همراه با سیستم.",
		SearchHint:     "برای جستجو تایپ کنید",
		Autoupdate:     "بروزرسانی خودکار",
		Expires:        "انقضا",
		Updated:        "بروزرسانی شده",
		Used:           "استفاده شده",
		Of:             "از",
		NoteLabel:      "یادداشت",
		NoSubs:         "اشتراکی یافت نشد",
		AddSubHint:     "«ctrl + a» را برای افزودن بزنید",

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
		Dead:           "بی‌پاسخ",
		NotFound:       "یافت نشد",

		SettingsAdvanced: "تنظیمات پیشرفته",
		SettingsSubs:     "اشتراک‌ها",
		SettingsReset:    "بازنشانی",
		SettingsRoutes:   "مسیرهای مستثنا",
		SetSearchHint:    "جستجو در تنظیمات",

		SecGeneral:       "عمومی",
		SecConnection:    "اتصال",
		SecOther:         "سایر",
		SecUpdate:        "بروزرسانی",
		SecSending:       "ارسال داده",
		SecUserAgent:     "User agent",
		SecServerList:    "فهرست سرورها",
		SecOtherSettings: "سایر تنظیمات",
		SecCIDR:          "فهرست CIDR",
		SecLogging:       "ثبت گزارش",
		SecSources:       "منابع",
		SecStorage:       "نگهداری",

		LblLanguage:     "زبان",
		LblPingProto:    "پروتکل پینگ",
		LblPreferIP:     "نوع IP ترجیحی",
		LblFrag:         "قطعه‌قطعه‌سازی",
		LblFragPackets:  "بسته‌ها",
		LblFragLength:   "طول",
		LblFragInterval: "فاصله",
		LblMux:          "Mux",
		LblMuxConc:      "تعداد همزمان",
		LblLAN:          "اجازه اتصال از شبکه محلی",
		LblCurrentIP:    "IP فعلی",
		LblSocks:        "پراکسی SOCKS5",
		NoteFrag:        "دست‌دادن TLS را تکه می‌کند تا DPI آن را نشناسد.",
		NoteMux:         "چند جریان روی یک اتصال.",
		NoteLAN:         "دستگاه‌های دیگر می‌توانند از این رایانه استفاده کنند.",

		LblAutoUpdate:  "بروزرسانی خودکار",
		LblUpdateEvery: "فاصله بروزرسانی (ساعت)",
		LblTimeout:     "مهلت درخواست",
		NoteUpdate:     "هر چند وقت اشتراک‌ها هنگام اجرا بروزرسانی شوند.",
		LblUpdateOpen:  "بروزرسانی هنگام باز شدن",
		LblPingOpen:    "پینگ هنگام باز شدن",
		LblConnectOpen: "اتصال هنگام باز شدن",
		NoteOnOpen:     "در هر بار اجرا انجام می‌شود.",
		LblNoDupes:     "نادیده گرفتن تکراری‌ها",
		NoteNoDupes:    "سرورهایی که از قبل در فهرست هستند را پنهان می‌کند.",
		LblSendHWID:    "ارسال HWID",
		NoteHWID:       "X-Hwid، X-Device-Os و X-Device-Model را می‌فرستد. شناسه از سیستم می‌آید و تغییر نمی‌کند.",
		NoteUserAgent:  "برای بازگشت به پیش‌فرض، فیلد را خالی کنید.",

		LblLocalDNS:    "استفاده از DNS محلی",
		LblJSONDNS:     "استفاده از DNS داخل JSON",
		LblResolveSrv:  "فعال‌سازی Resolve Server",
		LblSniffing:    "فعال‌سازی sniffing",
		LblSysProxy:    "پراکسی سیستمی",
		LblTUN:         "TUN",
		LblTunProvider: "موتور TUN",
		LblTunMode:     "حالت TUN",
		LblTunConfig:   "پیکربندی TUN",
		LblTunDNSOn:    "فعال‌سازی TUN DNS",
		LblTunDNS:      "آدرس TUN DNS",

		AddCIDR:  "افزودن CIDR…",
		NoteCIDR: "این مسیرها از تونل عبور نمی‌کنند و مستقیم می‌روند.",

		ResetUser:      "بازنشانی داده‌های کاربر",
		NoteResetUser:  "همه داده‌های کاربر را حذف می‌کند.",
		ResetPrefs:     "بازنشانی تنظیمات",
		NoteResetPrefs: "تنظیمات را بازنشانی می‌کند، اشتراک‌ها و سرورها می‌مانند.",
		ResetTun:       "بازنشانی پیکربندی تونل",
		NoteResetTun:   "همه رابط‌های tun ساخته‌شده را پاک می‌کند.",
		ConfirmTitle:   "مطمئنید؟",
		ConfirmBtn:     "بازنشانی",
		CancelBtn:      "انصراف",

		OptAuto:      "خودکار",
		OptIPv4:      "ipv4",
		OptIPv6:      "ipv6",
		OptTCP:       "tcp/ip",
		OptICMP:      "icmp",
		OptMixed:     "ترکیبی",
		OptSystem:    "سیستمی",
		OptGvisor:    "gvisor",
		OptDefault:   "پیش‌فرض",
		OptCustom:    "سفارشی",
		OptSingbox:   "sing-box",
		OptUnsorted:  "بدون مرتب‌سازی",
		OptSortPing:  "مرتب‌سازی با پینگ",
		OptSortAlpha: "مرتب‌سازی الفبایی",

		LblLogOn:      "فعال‌سازی گزارش",
		NoteLogOn:     "وقتی خاموش باشد، هیچ چیز ثبت نمی‌شود.",
		LblLogDaemon:  "رویدادهای دیمن",
		LblLogXray:    "Xray",
		LblLogSingbox: "Sing-box",
		NoteLogSrc:    "کدام منابع در گزارش ثبت شوند.",
		LblLogMax:     "بیشینه اندازه گزارش",
		NoteLogMax:    "۰ = بدون محدودیت. مثل 1 mb، 512 kb، 8 gb بنویسید.",
		OptNoLimit:    "بدون محدودیت",

		LogsFilterHint: "فیلتر",
		LogsEmpty:      "هنوز چیزی ثبت نشده",
		LogsFollow:     "دنبال‌کردن",
		LogsPaused:     "متوقف",
		LogsLines:      "خط",
		LogsHint:       "ctrl + e به انتها • c پاک‌کردن",

		Seconds: "ثانیه",
		Hours:   "ساعت",

		ConnLost:         "اتصال قطع شد",
		ErrCrashLoop:     "هسته مدام بسته می‌شود — سرور دیگری را امتحان کنید",
		ErrAutostart:     "تغییر اجرای خودکار ناموفق بود",
		ErrAutostartRoot: "اجرای خودکار برای کل سیستم تنظیم شده — برای تغییر، alexandria را با sudo اجرا کنید",
		ErrBadCIDR:       "این شبیه CIDR یا دامنه نیست",
		ErrBadSize:       "اندازه را مثل 1 mb، 512 kb یا 8 gb بنویسید",
		ErrBadPort:       "پورت باید بین ۱ تا ۶۵۵۳۴ باشد",
		ErrReset:         "بازنشانی ناموفق بود",
		OkReset:          "انجام شد",
	},
}

func T(code string) Strings {
	if s, ok := table[code]; ok {
		return s
	}
	return table["en"]
}
