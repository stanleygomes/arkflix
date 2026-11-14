; Instalador do arkflix pro Windows (NSIS). Instala na pasta do usuário
; (sem pedir administrador), cria atalhos no Menu Iniciar e na área de
; trabalho e registra o desinstalador. Chamado por bin/package.sh windows.
Unicode true
!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif

Name "arkflix"
OutFile "${OUT}"
InstallDir "$LOCALAPPDATA\Programs\arkflix"
InstallDirRegKey HKCU "Software\arkflix" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "arkflix"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "arkflix: jogos retrô, filmes e séries"
VIAddVersionKey "LegalCopyright" "arkflix"

!define MUI_FINISHPAGE_RUN "$INSTDIR\arkflix.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Abrir o arkflix"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "PortugueseBR"

Section "arkflix"
  ; antes o app se chamava arkflix-games: tira a instalação antiga (os dados
  ; em %AppData% o próprio app migra na primeira vez que abre)
  ReadRegStr $0 HKCU "Software\arkflix-games" "InstallDir"
  StrCmp $0 "" +4
  Delete "$0\arkflix-games.exe"
  Delete "$0\desinstalar.exe"
  RMDir "$0"
  Delete "$SMPROGRAMS\arkflix games\arkflix games.lnk"
  RMDir "$SMPROGRAMS\arkflix games"
  Delete "$DESKTOP\arkflix games.lnk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\arkflix-games"
  DeleteRegKey HKCU "Software\arkflix-games"
  SetOutPath "$INSTDIR"
  File "${SRC}\arkflix.exe"
  WriteUninstaller "$INSTDIR\desinstalar.exe"
  CreateDirectory "$SMPROGRAMS\arkflix"
  CreateShortcut "$SMPROGRAMS\arkflix\arkflix.lnk" "$INSTDIR\arkflix.exe"
  CreateShortcut "$DESKTOP\arkflix.lnk" "$INSTDIR\arkflix.exe"
  WriteRegStr HKCU "Software\arkflix" "InstallDir" "$INSTDIR"
  !define UNINST "Software\Microsoft\Windows\CurrentVersion\Uninstall\arkflix"
  WriteRegStr HKCU "${UNINST}" "DisplayName" "arkflix"
  WriteRegStr HKCU "${UNINST}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINST}" "DisplayIcon" "$INSTDIR\arkflix.exe"
  WriteRegStr HKCU "${UNINST}" "UninstallString" "$\"$INSTDIR\desinstalar.exe$\""
  WriteRegDWORD HKCU "${UNINST}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST}" "NoRepair" 1
SectionEnd

; Desinstalar tira o programa; configurações, perfis e saves
; (%AppData%\arkflix) ficam.
Section "Uninstall"
  Delete "$INSTDIR\arkflix.exe"
  Delete "$INSTDIR\desinstalar.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\arkflix\arkflix.lnk"
  RMDir "$SMPROGRAMS\arkflix"
  Delete "$DESKTOP\arkflix.lnk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\arkflix"
  DeleteRegKey HKCU "Software\arkflix"
SectionEnd
