package i18n

import "strings"

const (
	LocaleEnglish   = "en"
	LocaleUkrainian = "uk"
)

var embedCatalog = map[string]map[string]string{
	LocaleEnglish: {
		"best":                        "Best",
		"anonymous":                   "Anonymous",
		"cancel":                      "Cancel",
		"comment":                     "Comment",
		"comment_placeholder":         "Add your comment here...",
		"comment_posted":              "Comment posted.",
		"comment_submitted":           "Comment submitted.",
		"comments":                    "Comments",
		"comments_archived":           "This discussion is archived.",
		"comments_hidden":             "Comments are hidden.",
		"comments_locked":             "Comments are locked.",
		"comments_not_configured":     "Comments are not configured.",
		"comments_unavailable":        "Comments are unavailable.",
		"comments_unavailable_origin": "Comments are unavailable on this origin.",
		"comment_too_long":            "Comment is too long.",
		"email_optional_not_shown":    "Email (optional, not shown)",
		"name":                        "Name",
		"invalid_json":                "Invalid JSON.",
		"markdown_supported":          "Markdown is supported",
		"newest":                      "Newest",
		"no_comments_yet":             "No comments yet.",
		"nothing_to_preview":          "Nothing to preview.",
		"not_posted":                  "Comment was not posted.",
		"origin_not_allowed":          "Origin is not allowed for this site.",
		"parent_invalid":              "Reply target is invalid.",
		"page_posting_closed":         "This page does not allow new comments.",
		"oldest":                      "Oldest",
		"pending_message":             "Comment submitted and waiting for moderation.",
		"pending_review":              "pending review",
		"pending_rule_message":        "Comment submitted and waiting for moderation because it matched a moderation rule.",
		"preview":                     "Preview",
		"preview_unavailable":         "Preview unavailable.",
		"rejected_default":            "Comment rejected by moderation.",
		"rejected_ip_banned":          "Comment rejected: this network is blocked for this site.",
		"rejected_rate_limit":         "Comment rejected: too many comments were submitted recently. Please try again later.",
		"rejected_rate_limit_retry":   "Comment rejected: too many comments were submitted recently. Try again in about %s.",
		"rejected_word_ban":           "Comment rejected by this site's moderation rules.",
		"rendering_preview":           "Rendering preview...",
		"reply":                       "Reply",
		"reply_to":                    "Reply to",
		"replying_to":                 "replying to @",
		"sort_comments":               "Sort comments",
		"spam_default":                "Comment rejected by spam protection.",
		"spam_duplicate":              "Comment rejected by spam protection: duplicate comment.",
		"spam_honeypot":               "Comment rejected by spam protection.",
		"spam_links":                  "Comment rejected by spam protection: too many links.",
		"spam_word_ban":               "Comment rejected by this site's moderation rules.",
		"submit_comment":              "Submit comment",
		"required_body":               "Comment is required.",
		"required_name":               "Name is required.",
		"reserved_name":               "This name is reserved.",
		"replies_disabled":            "Replies are disabled for this site.",
		"submit_reply":                "Submit reply",
		"submitting":                  "Submitting...",
		"time_just_now":               "just now",
		"tripcode_help":               "A tripcode is a public, stable marker generated from a secret, so others can recognize the same anonymous commenter without an account. Reserved names require the correct secret and may show a verified badge.",
		"tripcode_help_label":         "What is a tripcode?",
		"unable_submit":               "Unable to submit comment. Check your connection and try again.",
		"use_tripcode":                "Use Name##secret for a tripcode",
		"used_only_for_avatar":        "Used only for avatar",
		"website_optional":            "Website (optional)",
		"write":                       "Write",
	},
	LocaleUkrainian: {
		"best":                        "Найкращі",
		"anonymous":                   "Анонім",
		"cancel":                      "Скасувати",
		"comment":                     "Коментар",
		"comment_placeholder":         "Напишіть коментар...",
		"comment_posted":              "Коментар опубліковано.",
		"comment_submitted":           "Коментар надіслано.",
		"comments":                    "Коментарі",
		"comments_archived":           "Це обговорення заархівоване.",
		"comments_hidden":             "Коментарі приховані.",
		"comments_locked":             "Коментарі закриті.",
		"comments_not_configured":     "Коментарі не налаштовані.",
		"comments_unavailable":        "Коментарі недоступні.",
		"comments_unavailable_origin": "Коментарі недоступні для цього origin.",
		"comment_too_long":            "Коментар занадто довгий.",
		"email_optional_not_shown":    "Ел. пошта (необовʼязково, не показується)",
		"name":                        "Імʼя",
		"invalid_json":                "Некоректний JSON.",
		"markdown_supported":          "Markdown підтримується",
		"newest":                      "Нові",
		"no_comments_yet":             "Коментарів ще немає.",
		"nothing_to_preview":          "Немає що показати.",
		"not_posted":                  "Коментар не опубліковано.",
		"origin_not_allowed":          "Origin не дозволений для цього сайту.",
		"parent_invalid":              "Некоректна ціль відповіді.",
		"page_posting_closed":         "На цій сторінці не можна додавати нові коментарі.",
		"oldest":                      "Старі",
		"pending_message":             "Коментар надіслано й очікує модерації.",
		"pending_review":              "очікує модерації",
		"pending_rule_message":        "Коментар надіслано й очікує модерації, бо спрацювало правило модерації.",
		"preview":                     "Перегляд",
		"preview_unavailable":         "Перегляд недоступний.",
		"rejected_default":            "Коментар відхилено модерацією.",
		"rejected_ip_banned":          "Коментар відхилено: ця мережа заблокована для цього сайту.",
		"rejected_rate_limit":         "Коментар відхилено: забагато коментарів за короткий час. Спробуйте пізніше.",
		"rejected_rate_limit_retry":   "Коментар відхилено: забагато коментарів за короткий час. Спробуйте приблизно через %s.",
		"rejected_word_ban":           "Коментар відхилено правилами модерації цього сайту.",
		"rendering_preview":           "Рендеримо перегляд...",
		"reply":                       "Відповісти",
		"reply_to":                    "Відповісти",
		"replying_to":                 "відповідь @",
		"sort_comments":               "Сортування коментарів",
		"spam_default":                "Коментар відхилено spam-захистом.",
		"spam_duplicate":              "Коментар відхилено spam-захистом: дублікат.",
		"spam_honeypot":               "Коментар відхилено spam-захистом.",
		"spam_links":                  "Коментар відхилено spam-захистом: забагато посилань.",
		"spam_word_ban":               "Коментар відхилено правилами модерації цього сайту.",
		"submit_comment":              "Надіслати коментар",
		"required_body":               "Коментар обовʼязковий.",
		"required_name":               "Імʼя обовʼязкове.",
		"reserved_name":               "Це імʼя зарезервоване.",
		"replies_disabled":            "Відповіді вимкнені для цього сайту.",
		"submit_reply":                "Надіслати відповідь",
		"submitting":                  "Надсилаємо...",
		"time_just_now":               "щойно",
		"tripcode_help":               "Tripcode — це публічна стабільна мітка з вашого секрету, щоб інші впізнавали того самого анонімного коментатора без акаунта. Зарезервовані імена потребують правильного секрету й можуть мати verified badge.",
		"tripcode_help_label":         "Що таке tripcode?",
		"unable_submit":               "Не вдалося надіслати коментар. Перевірте зʼєднання і спробуйте ще раз.",
		"use_tripcode":                "Name##secret для tripcode",
		"used_only_for_avatar":        "Використовується тільки для аватарки",
		"website_optional":            "Сайт (необовʼязково)",
		"write":                       "Написати",
	},
}

func Normalize(raw string, acceptLanguage string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		raw = strings.ToLower(strings.TrimSpace(firstLanguage(acceptLanguage)))
	}
	if strings.HasPrefix(raw, "uk") {
		return LocaleUkrainian
	}
	return LocaleEnglish
}

func Embed(locale string) map[string]string {
	normalized := Normalize(locale, "")
	base := embedCatalog[LocaleEnglish]
	selected := embedCatalog[normalized]
	out := make(map[string]string, len(base))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range selected {
		out[key] = value
	}
	return out
}

func Text(locale string, key string) string {
	catalog := Embed(locale)
	if value := catalog[key]; value != "" {
		return value
	}
	return key
}

func firstLanguage(header string) string {
	if header == "" {
		return ""
	}
	part := strings.Split(header, ",")[0]
	return strings.TrimSpace(strings.Split(part, ";")[0])
}
