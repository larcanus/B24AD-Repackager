package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type Manifest struct {
	Name                   string                   `json:"name"`
	Version                string                   `json:"version"`
	ContentScripts         []ContentScript          `json:"content_scripts"`
	WebAccessibleResources []WebAccessibleResource  `json:"web_accessible_resources"`
	HostPermissions        []string                 `json:"host_permissions"`
	Permissions            []string                 `json:"permissions"`
	ManifestVersion        int                      `json:"manifest_version"`
}

type ContentScript struct {
	Matches []string `json:"matches"`
	JS      []string `json:"js"`
	CSS     []string `json:"css"`
}

type WebAccessibleResource struct {
	Matches []string `json:"matches"`
	Resources []string `json:"resources"`
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Bitrix24 Bad Advice Extension Repackager")
	myWindow.Resize(fyne.NewSize(600, 500))

	var inputFile string
	var tempDir string
	var manifestData Manifest

	// Создаем основной интерфейс
	instructions := widget.NewLabel(`Bitrix24 Bad Advice Extension Repackager

Эта программа поможет пересобрать расширение Chrome для Bitrix24 Bad Advice .

Шаги:
1. Выберите архив расширения (.zip)
2. Программа проверит валидность архива
3. Введите URL вашего портала Bitrix24
4. Выберите куда сохранить пересобранное расширение`)
	instructions.Wrapping = fyne.TextWrapWord

	fileLabel := widget.NewLabel("Выберите файл расширения:")
	fileEntry := widget.NewEntry()
	fileEntry.SetPlaceHolder("Выберите ZIP файл расширения...")

	fileEntry.OnChanged = func(text string) {
		inputFile = text
	}

	browseButton := widget.NewButton("Обзор...", func() {
		dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				inputFile = reader.URI().Path()
				fileEntry.SetText(inputFile)
			}
		}, myWindow).Show()
	})

	nextButton := widget.NewButton("Далее", func() {
		validateArchive(myWindow, inputFile, &tempDir, &manifestData)
	})

	content := container.NewVBox(
		instructions,
		fileLabel,
		container.NewHBox(fileEntry, browseButton),
		layout.NewSpacer(),
		nextButton,
	)

	myWindow.SetContent(content)
	myWindow.ShowAndRun()
}

func validateArchive(window fyne.Window, inputFile string, tempDir *string, manifestData *Manifest) {
	if inputFile == "" {
		showError(window, "Файл не выбран")
		return
	}

	// Создаем временную директорию
	*tempDir = filepath.Join(os.TempDir(), "bitrix_repack")
	os.RemoveAll(*tempDir)
	os.MkdirAll(*tempDir, 0755)

	// Распаковываем архив
	err := unzip(inputFile, *tempDir)
	if err != nil {
		showError(window, "Ошибка распаковки: "+err.Error())
		return
	}

	// Определяем корневую папку расширения
	rootDir, err := findExtensionRoot(*tempDir)
	if err != nil {
		showError(window, err.Error())
		return
	}

	// Проверяем manifest.json
	manifestPath := filepath.Join(rootDir, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		showError(window, "manifest.json не найден: "+err.Error())
		return
	}

	// Парсим manifest.json
	err = json.Unmarshal(manifestBytes, manifestData)
	if err != nil {
		showError(window, "Ошибка чтения manifest.json: "+err.Error())
		return
	}

	// Проверяем что это Bitrix24
	if manifestData.Name != "Bitrix24: Bad Advice" {
		showError(window, "Это не расширение Bitrix24")
		return
	}

	// Обновляем tempDir на корневую папку расширения
	*tempDir = rootDir

	// Показываем следующий экран
	showURLInput(window, *tempDir, *manifestData)
}

// Функция для поиска корневой папки расширения
func findExtensionRoot(dir string) (string, error) {
	// Проверяем, есть ли manifest.json прямо в корне
	rootManifest := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(rootManifest); err == nil {
		return dir, nil
	}

	// Ищем во вложенных папках
	var manifestPath string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "manifest.json" {
			manifestPath = path
			return filepath.SkipAll
		}
		return nil
	})

	if manifestPath == "" {
		return "", fmt.Errorf("manifest.json не найден в архиве")
	}

	// Возвращаем папку, содержащую manifest.json
	return filepath.Dir(manifestPath), nil
}

func showURLInput(window fyne.Window, tempDir string, manifestData Manifest) {
	urlLabel := widget.NewLabel("URL вашего портала Bitrix24:")
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("например: mycompany.bitrix24.ru или crm.mycompany.com")

	buildButton := widget.NewButton("Собрать", func() {
		repackExtension(window, urlEntry.Text, tempDir, manifestData)
	})

	content := container.NewVBox(
		widget.NewLabel("Введите URL вашего портала Bitrix24"),
		urlLabel,
		urlEntry,
		layout.NewSpacer(),
		container.NewHBox(
			widget.NewButton("Назад", func() {
				os.RemoveAll(tempDir)
				// Для простоты перезапускаем приложение
				os.Exit(0)
			}),
			buildButton,
		),
	)

	window.SetContent(content)
}

func repackExtension(window fyne.Window, urlInput string, tempDir string, manifestData Manifest) {
	urlInput = strings.TrimSpace(urlInput)
	if urlInput == "" {
		showError(window, "Введите URL портала")
		return
	}

	httpURL, httpsURL := normalizeURL(urlInput)

	// Обновляем манифест
	updateManifest(&manifestData, httpURL, httpsURL)

	// Сохраняем обновленный манифест
	manifestPath := filepath.Join(tempDir, "manifest.json") // ← используем tempDir который теперь указывает на корень расширения
	manifestBytes, err := json.MarshalIndent(manifestData, "", "  ")
	if err != nil {
		showError(window, "Ошибка сохранения manifest.json: "+err.Error())
		return
	}

	err = os.WriteFile(manifestPath, manifestBytes, 0644)
	if err != nil {
		showError(window, "Ошибка записи manifest.json: "+err.Error())
		return
	}

	// Сохраняем новый архив (архивируем корневую папку расширения)
	dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err == nil && writer != nil {
			defer writer.Close()

			outputFile := writer.URI().Path()
			if !strings.HasSuffix(outputFile, ".zip") {
				outputFile += ".zip"
			}

			err := zipDirectory(tempDir, outputFile)
			if err != nil {
				showError(window, "Ошибка создания архива: "+err.Error())
				return
			}

			dialog.ShowInformation("Успех",
				fmt.Sprintf("Расширение успешно пересобрано!\n\nДобавленные URL:\n%s\n%s\n\nСохранено в: %s",
					httpURL, httpsURL, outputFile), window)

			// Очищаем временную папку (удаляем всю tempDir, а не только корень расширения)
			os.RemoveAll(filepath.Dir(tempDir))
		}
	}, window).Show()
}

func normalizeURL(input string) (string, string) {
	// Убираем протокол если есть
	re := regexp.MustCompile(`^https?://`)
	input = re.ReplaceAllString(input, "")

	// Убираем слэши в конце
	input = strings.TrimRight(input, "/")

	return "http://" + input, "https://" + input
}

func showError(window fyne.Window, message string) {
	dialog.ShowError(fmt.Errorf(message), window)
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), 0755)
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func zipDirectory(source, target string) error {
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(source, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

func updateManifest(manifest *Manifest, httpURL, httpsURL string) {
	httpPattern := httpURL + "/*"
	httpsPattern := httpsURL + "/*"

	// Обновляем content_scripts
	for i := range manifest.ContentScripts {
		manifest.ContentScripts[i].Matches = []string{httpPattern, httpsPattern}
	}

	// Обновляем web_accessible_resources
	for i := range manifest.WebAccessibleResources {
		manifest.WebAccessibleResources[i].Matches = []string{httpPattern, httpsPattern}
	}

	// Обновляем host_permissions
	if len(manifest.HostPermissions) > 0 {
		manifest.HostPermissions = []string{httpPattern, httpsPattern}
	}

	// Обновляем permissions
	var newPerms []string
	for _, perm := range manifest.Permissions {
		if !strings.Contains(perm, "://") {
			newPerms = append(newPerms, perm)
		}
	}
	newPerms = append(newPerms, httpPattern, httpsPattern)
	manifest.Permissions = newPerms
}
