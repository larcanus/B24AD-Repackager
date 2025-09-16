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
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/storage"
)

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("B24AD Extension Repackager")
	myWindow.Resize(fyne.NewSize(650, 520))

	var inputFile string
	var tempDir string
	var manifestData map[string]interface{}

	// Обёртка для возврата на первый шаг
	var showStep1Func func()
	showStep1Func = func() {
		showStep1(myWindow, &inputFile, &tempDir, &manifestData, showStep1Func)
	}

	showStep1Func()
	myWindow.ShowAndRun()
}

// Первый шаг: выбор файла
func showStep1(window fyne.Window, inputFile *string, tempDir *string, manifestData *map[string]interface{}, showStep1Func func()) {
	title := widget.NewLabelWithStyle("Bitrix24 Bad Advice Extension Repackager", fyne.TextAlignCenter, fyne.TextStyle{Bold: true, Monospace: false})

	instructions := widget.NewRichTextFromMarkdown(`
**С помощью этого приложения вы можете пересобрать расширение B24AD с уточнением уникальных адресов для вашего Bitrix24.**
`)
	steps := widget.NewRichTextFromMarkdown(`
**Шаги:**
1. _Выберите архив расширения (.zip)_
2. _Введите URL вашего портала Bitrix24_
3. _Выберите куда сохранить_`)
	instructions.Wrapping = fyne.TextWrapWord
	steps.Wrapping = fyne.TextWrapWord

	fileLabel := widget.NewLabelWithStyle("Выбор архива расширения:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	fileEntry := widget.NewEntry()
	fileEntry.SetPlaceHolder("ZIP файл...")
	fileEntry.Resize(fyne.NewSize(380, fileEntry.MinSize().Height))

	fileEntry.OnChanged = func(text string) {
		*inputFile = text
	}

	browseButton := widget.NewButtonWithIcon("Обзор...", theme.FolderOpenIcon(), func() {
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				*inputFile = reader.URI().Path()
				fileEntry.SetText(*inputFile)
			}
		}, window)
		fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".zip"}))
		fileDialog.Resize(fyne.NewSize(700, 500))
		fileDialog.Show()
	})
	browseButton.Resize(fyne.NewSize(110, browseButton.MinSize().Height))

	fileRow := container.NewGridWithColumns(2,
		container.NewMax(fileEntry),
		browseButton,
	)

	nextButton := widget.NewButtonWithIcon("Далее", theme.NavigateNextIcon(), func() {
		validateArchive(window, *inputFile, tempDir, manifestData, showStep1Func)
	})
	nextButton.Importance = widget.HighImportance

	content := container.NewVBox(
		title,
		layout.NewSpacer(),
		widget.NewLabel("\n"),
		instructions,
		layout.NewSpacer(),
		steps,
		layout.NewSpacer(),
		widget.NewLabel("\n"),
		fileLabel,
		fileRow,
		layout.NewSpacer(),
		widget.NewLabel("\n"),
		container.NewCenter(nextButton),
	)

	padded := container.NewVBox(
		layout.NewSpacer(),
		container.NewVBox(
			layout.NewSpacer(),
			content,
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
	)
	window.SetContent(container.NewPadded(padded))
}

func validateArchive(window fyne.Window, inputFile string, tempDir *string, manifestData *map[string]interface{}, backToStep1 func()) {
	if inputFile == "" {
		showError(window, "Файл не выбран")
		return
	}

	*tempDir = filepath.Join(os.TempDir(), "bitrix_repack")
	os.RemoveAll(*tempDir)
	os.MkdirAll(*tempDir, 0755)

	err := unzip(inputFile, *tempDir)
	if err != nil {
		showError(window, "Ошибка распаковки: "+err.Error())
		return
	}

	rootDir, err := findExtensionRoot(*tempDir)
	if err != nil {
		showError(window, err.Error())
		return
	}

	manifestPath := filepath.Join(rootDir, "manifest.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		showError(window, "manifest.json не найден: "+err.Error())
		return
	}

	var manifest map[string]interface{}
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		showError(window, "Ошибка чтения manifest.json: "+err.Error())
		return
	}

	// Проверяем имя расширения
	name, ok := manifest["name"].(string)
	if !ok || name != "Bitrix24: Bad Advice" {
		showError(window, "Это не расширение Bitrix24")
		return
	}

	*manifestData = manifest
	*tempDir = rootDir

	showURLInput(window, *tempDir, *manifestData, backToStep1)
}

func findExtensionRoot(dir string) (string, error) {
	rootManifest := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(rootManifest); err == nil {
		return dir, nil
	}

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

	return filepath.Dir(manifestPath), nil
}

func showURLInput(window fyne.Window, tempDir string, manifestData map[string]interface{}, backToStep1 func()) {
	title := widget.NewLabelWithStyle("Ввод нового URL", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	urlLabel := widget.NewLabelWithStyle("URL вашего портала Bitrix24:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("например: mycompany.bitrix24.ru или crm.mycompany.com")
	urlEntry.Resize(fyne.NewSize(380, urlEntry.MinSize().Height))

	buildButton := widget.NewButtonWithIcon("Собрать", theme.ConfirmIcon(), func() {
		repackExtension(window, urlEntry.Text, tempDir, manifestData)
	})
	buildButton.Importance = widget.HighImportance

	backButton := widget.NewButtonWithIcon("Назад", theme.NavigateBackIcon(), func() {
		backToStep1()
	})

	buttons := container.NewHBox(
		layout.NewSpacer(),
		backButton,
		layout.NewSpacer(),
		buildButton,
		layout.NewSpacer(),
	)

	content := container.NewVBox(
		title,
		layout.NewSpacer(),
		urlLabel,
		urlEntry,
		layout.NewSpacer(),
		buttons,
	)

	window.SetContent(container.NewPadded(content))
}

func repackExtension(window fyne.Window, urlInput string, tempDir string, manifestData map[string]interface{}) {
	urlInput = strings.TrimSpace(urlInput)
	if urlInput == "" {
		showError(window, "Введите URL портала")
		return
	}

	httpURL, httpsURL := normalizeURL(urlInput)

	updateManifestMap(manifestData, httpURL, httpsURL)

	manifestPath := filepath.Join(tempDir, "manifest.json")
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

	defaultName := "B24AD-custom-url.zip"
	dlg := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
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

			// Показываем диалог успеха и закрываем приложение после OK
			successDlg := dialog.NewCustom("Успех", "OK",
				widget.NewLabel(fmt.Sprintf("Расширение успешно пересобрано!\n\nДобавленные URL:\n%s\n%s\n\nСохранено в: %s",
					httpURL, httpsURL, outputFile)),
				window,
			)
			successDlg.SetOnClosed(func() {
				window.Close()
			})
			successDlg.Show()

			os.RemoveAll(filepath.Dir(tempDir))
		}
	}, window)
	dlg.SetFileName(defaultName)
	dlg.Resize(fyne.NewSize(700, 500))
	dlg.Show()
}

func normalizeURL(input string) (string, string) {
	re := regexp.MustCompile(`^https?://`)
	input = re.ReplaceAllString(input, "")
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

// Гибкая модификация manifest.json через map[string]interface{}
func updateManifestMap(manifest map[string]interface{}, httpURL, httpsURL string) {
	httpPattern := httpURL + "/*"
	httpsPattern := httpsURL + "/*"

	addUnique := func(slice []interface{}, values ...string) []interface{} {
		exists := make(map[string]struct{}, len(slice))
		for _, v := range slice {
			if str, ok := v.(string); ok {
				exists[str] = struct{}{}
			}
		}
		for _, v := range values {
			if _, ok := exists[v]; !ok {
				slice = append(slice, v)
				exists[v] = struct{}{}
			}
		}
		return slice
	}

	// content_scripts
	if csVal, ok := manifest["content_scripts"].([]interface{}); ok {
		for i, cs := range csVal {
			if csMap, ok := cs.(map[string]interface{}); ok {
				if matches, ok := csMap["matches"].([]interface{}); ok {
					csMap["matches"] = addUnique(matches, httpPattern, httpsPattern)
				} else {
					csMap["matches"] = []interface{}{httpPattern, httpsPattern}
				}
				csVal[i] = csMap
			}
		}
		manifest["content_scripts"] = csVal
	}

	// web_accessible_resources
	if waVal, ok := manifest["web_accessible_resources"].([]interface{}); ok {
		for i, wa := range waVal {
			if waMap, ok := wa.(map[string]interface{}); ok {
				if matches, ok := waMap["matches"].([]interface{}); ok {
					waMap["matches"] = addUnique(matches, httpPattern, httpsPattern)
				} else {
					waMap["matches"] = []interface{}{httpPattern, httpsPattern}
				}
				waVal[i] = waMap
			}
		}
		manifest["web_accessible_resources"] = waVal
	}

	// host_permissions
	if hpVal, ok := manifest["host_permissions"].([]interface{}); ok {
		manifest["host_permissions"] = addUnique(hpVal, httpPattern, httpsPattern)
	} else {
		manifest["host_permissions"] = []interface{}{httpPattern, httpsPattern}
	}

	// permissions (если нужно)
	// if pVal, ok := manifest["permissions"].([]interface{}); ok {
	// 	manifest["permissions"] = addUnique(pVal, httpPattern, httpsPattern)
	// }
}
