!define APP_NAME "PowerScheduler"
!define APP_EXE "power-desktop.exe"
!define OUT_FILE "..\release\powerschedulersetup.exe"
!define INSTALL_DIR "$LOCALAPPDATA\PowerScheduler"

Name "${APP_NAME}"
OutFile "${OUT_FILE}"
InstallDir "${INSTALL_DIR}"
RequestExecutionLevel user

Page directory
Page instfiles

Section "Install"
  SetOutPath "$INSTDIR"
  File "..\dist\${APP_EXE}"
  SetOutPath "$INSTDIR\web"
  File /r "..\dist\web\*.*"

  CreateDirectory "$SMPROGRAMS\PowerScheduler"
  CreateShortCut "$SMPROGRAMS\PowerScheduler\PowerScheduler.lnk" "$INSTDIR\${APP_EXE}" "" "$INSTDIR\${APP_EXE}" 0 SW_SHOWNORMAL
  CreateShortCut "$DESKTOP\PowerScheduler.lnk" "$INSTDIR\${APP_EXE}" "" "$INSTDIR\${APP_EXE}" 0 SW_SHOWNORMAL

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
