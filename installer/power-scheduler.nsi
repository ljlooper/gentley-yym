!define APP_NAME "PowerScheduler"
!define APP_EXE "power-desktop.exe"
!define OUT_FILE "..\dist\PowerSchedulerSetup.exe"
!define INSTALL_DIR "$PROGRAMFILES\PowerScheduler"

Name "${APP_NAME}"
OutFile "${OUT_FILE}"
InstallDir "${INSTALL_DIR}"
RequestExecutionLevel admin

Page directory
Page instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  File "..\dist\${APP_EXE}"
  SetOutPath "$INSTDIR\web"
  File /r "..\dist\web\*.*"

  CreateDirectory "$SMPROGRAMS\PowerScheduler"
  CreateShortCut "$SMPROGRAMS\PowerScheduler\PowerScheduler.lnk" "$INSTDIR\${APP_EXE}"
  CreateShortCut "$DESKTOP\PowerScheduler.lnk" "$INSTDIR\${APP_EXE}"

  WriteUninstaller "$INSTDIR\Uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$SMPROGRAMS\PowerScheduler\PowerScheduler.lnk"
  RMDir "$SMPROGRAMS\PowerScheduler"
  Delete "$DESKTOP\PowerScheduler.lnk"
  RMDir /r "$INSTDIR\web"
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
