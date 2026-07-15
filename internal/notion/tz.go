package notion

import (
	"fmt"
	"regexp"
	"strings"
)

type TaskInfo struct {
	Name       string `json:"name"`
	Section    string `json:"section"`
	Priority   string `json:"priority"`
	Complexity string `json:"complexity"`
	TaskType   string `json:"task_type"`
	Notes      string `json:"notes"`
}

type AnalysisResult struct {
	Status        string     `json:"status"`
	Message       string     `json:"message,omitempty"`
	ProjectName   string     `json:"project_name"`
	Tasks         []TaskInfo `json:"tasks"`
	OriginalTasks []TaskInfo `json:"original_tasks,omitempty"`
	TotalTasks    int        `json:"total_tasks"`
	Summary       string     `json:"summary"`
}

var actionVerbs = []string{
	"создать", "разработать", "реализовать", "внедрить", "настроить", "установить",
	"протестировать", "проверить", "анализировать", "изучить", "исследовать",
	"спроектировать", "оптимизировать", "интегрировать", "автоматизировать",
	"документировать", "описать", "подготовить", "обучить", "провести",
}

func AnalyzeTechnicalRequirements(tz string) (*AnalysisResult, error) {
	lines := strings.Split(tz, "\n")

	projectName := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if projectName != "" {
			break
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "проект") || strings.Contains(lower, "название") ||
			strings.Contains(lower, "system") || strings.Contains(lower, "application") ||
			strings.Contains(lower, "project") {
			candidate := strings.ReplaceAll(strings.ReplaceAll(line, "#", ""), "*", "")
			candidate = strings.TrimSpace(candidate)
			if len(candidate) > 50 {
				candidate = candidate[:50] + "..."
			}
			projectName = candidate
		}
	}
	if projectName == "" {
		for _, line := range lines {
			if s := strings.TrimSpace(line); s != "" && !strings.HasPrefix(s, "#") {
				projectName = s[:minimal(len(s), 50)]
				break
			}
		}
	}

	tasks := make([]TaskInfo, 0)
	currentSection := ""
	listRe := regexp.MustCompile(`^\d+\.`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "#") || (strings.HasPrefix(line, "*") && len(line) > 10) {
			currentSection = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(line, "#", ""), "*", ""))
			continue
		}

		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || listRe.MatchString(line) {
			taskText := regexp.MustCompile(`^[-•\d.]\s*`).ReplaceAllString(line, "")
			taskText = strings.TrimSpace(taskText)
			if len(taskText) <= 10 {
				continue
			}
			taskText = enhanceTaskWithAI(taskText, currentSection)

			priority := "Medium Priority"
			tl := strings.ToLower(taskText)
			if containsAny(tl, "критичн", "срочн", "важн", "critical", "urgent", "high") {
				priority = "High Priority"
			} else if containsAny(tl, "низк", "опциональн", "желательн", "low", "optional") {
				priority = "Low Priority"
			}

			complexity := "Normal Complexity"
			if containsAny(tl, "сложн", "комплексн", "архитектур", "интеграц", "complex", "architecture") {
				complexity = "Hard Complexity"
			} else if containsAny(tl, "простой", "легк", "базов", "easy", "simple", "basic") {
				complexity = "Easy Complexity"
			}

			taskType := "PROJECT TASK"
			if containsAny(tl, "дизайн", "ui", "ux", "design") {
				taskType = "DESIGN"
			} else if containsAny(tl, "документ", "тз", "спецификац", "documentation", "spec") {
				taskType = "DOCUMENTATION"
			}

			notes := "Из раздела: " + currentSection
			if currentSection == "" {
				notes = "Извлечено из ТЗ"
			}

			tasks = append(tasks, TaskInfo{
				Name:       taskText,
				Section:    currentSection,
				Priority:   priority,
				Complexity: complexity,
				TaskType:   taskType,
				Notes:      notes,
			})
		}
	}

	if len(tasks) < 3 {
		defaultTasks := []TaskInfo{
			{
				Name:       "Провести анализ требований и планирование проекта",
				Section:    "Планирование",
				Priority:   "High Priority",
				Complexity: "Normal Complexity",
				TaskType:   "PROJECT TASK",
				Notes:      "Детальный анализ технического задания и планирование этапов",
			},
			{
				Name:       "Спроектировать архитектуру системы",
				Section:    "Архитектура",
				Priority:   "High Priority",
				Complexity: "Hard Complexity",
				TaskType:   "PROJECT TASK",
				Notes:      "Проектирование технической архитектуры на основе ТЗ",
			},
			{
				Name:       "Реализовать основной функционал системы",
				Section:    "Разработка",
				Priority:   "Medium Priority",
				Complexity: "Normal Complexity",
				TaskType:   "PROJECT TASK",
				Notes:      "Разработка ключевых функций и компонентов системы",
			},
		}
		tasks = append(tasks, defaultTasks...)
	}

	enhanced := enhanceTasksWithAI(tasks, orDefault(projectName, "Новый проект"))

	return &AnalysisResult{
		Status:        "success",
		ProjectName:   orDefault(projectName, "Новый проект"),
		Tasks:         enhanced,
		OriginalTasks: tasks,
		TotalTasks:    len(enhanced),
		Summary:       fmt.Sprintf("Проанализировано ТЗ: найдено %d задач для проекта '%s'", len(enhanced), orDefault(projectName, "Новый проект")),
	}, nil
}

func enhanceTasksWithAI(tasks []TaskInfo, projectContext string) []TaskInfo {
	enhanced := make([]TaskInfo, 0, len(tasks))
	for _, task := range tasks {
		enhancedName := enhanceTaskNameSmart(task.Name, task.Section, projectContext)
		e := task
		e.Name = enhancedName
		e.Notes = appendOriginal(e.Notes, task.Name)
		enhanced = append(enhanced, e)
	}
	return enhanced
}

func enhanceTaskWithAI(taskText, section string) string {
	return taskText
}

func enhanceTaskNameSmart(taskName, section, projectContext string) string {
	taskLower := strings.ToLower(taskName)

	for _, verb := range actionVerbs {
		if strings.HasPrefix(taskLower, verb) {
			return taskName
		}
	}

	lower := taskName
	switch {
	case containsAny(taskLower, "система", "модуль", "компонент", "сервис"):
		switch {
		case containsAny(taskLower, "аутентификации", "авторизации", "входа"):
			lower = "Разработать " + taskLower
		case containsAny(taskLower, "базы данных", "бд", "хранения"):
			lower = "Спроектировать " + taskLower
		case containsAny(taskLower, "уведомлений", "нотификаций"):
			lower = "Реализовать " + taskLower
		default:
			lower = "Создать " + taskLower
		}
	case containsAny(taskLower, "интеграция", "подключение", "связь"):
		switch {
		case containsAny(taskLower, "api", "апи"):
			lower = "Реализовать " + taskLower
		case containsAny(taskLower, "git", "гит", "slack", "telegram"):
			lower = "Настроить " + taskLower
		default:
			lower = "Интегрировать " + taskLower
		}
	case containsAny(taskLower, "тестирование", "тесты", "проверка"):
		switch {
		case containsAny(taskLower, "unit", "юнит"):
			lower = "Написать " + taskLower
		case containsAny(taskLower, "интеграционн"):
			lower = "Провести " + taskLower
		default:
			lower = "Выполнить " + taskLower
		}
	case containsAny(taskLower, "документация", "описание", "спецификация", "руководство"):
		switch {
		case containsAny(taskLower, "api"):
			lower = "Подготовить " + taskLower
		case containsAny(taskLower, "пользователь"):
			lower = "Написать " + taskLower
		default:
			lower = "Создать " + taskLower
		}
	case containsAny(taskLower, "настройка", "конфигурация", "установка"):
		if containsAny(taskLower, "сервер", "среда") {
			lower = "Настроить " + taskLower
		} else {
			lower = "Установить " + taskLower
		}
	case containsAny(taskLower, "ui", "ux", "дизайн", "интерфейс"):
		if containsAny(taskLower, "макет", "прототип") {
			lower = "Создать " + taskLower
		} else {
			lower = "Разработать " + taskLower
		}
	case containsAny(taskLower, "архитектура", "структура", "схема"):
		lower = "Спроектировать " + taskLower
	case containsAny(taskLower, "анализ", "исследование", "изучение"):
		lower = "Провести " + taskLower
	case containsAny(taskLower, "обучение", "подготовка"):
		if containsAny(taskLower, "пользователь") {
			lower = "Организовать " + taskLower
		} else {
			lower = "Подготовить " + taskLower
		}
	}

	if lower == taskName {
		sectionLower := strings.ToLower(section)
		switch {
		case containsAny(sectionLower, "требования"):
			lower = "Определить " + taskLower
		case containsAny(sectionLower, "функционал"):
			lower = "Реализовать " + taskLower
		case containsAny(sectionLower, "тестирование"):
			lower = "Провести " + taskLower
		case containsAny(sectionLower, "развертывание", "внедрение"):
			lower = "Выполнить " + taskLower
		default:
			lower = "Выполнить " + taskLower
		}
	}

	return lower
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func appendOriginal(notes, original string) string {
	if notes == "" {
		return "Оригинал: " + original
	}
	return notes + " (Оригинал: " + original + ")"
}

func minimal(a, b int) int {
	if a < b {
		return a
	}
	return b
}
