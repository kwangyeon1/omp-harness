# OMP Harness

## 스크립트·플러그인을 선택 실행하는 배포 묶음 (0.2.0)

`.exe`·`.run`은 관리 도구와 개별 스크립트를 배치합니다. OMP/플러그인 설치를 자동으로 연속 실행하지 않습니다.
관리 도구는 `catalog.json` 목록과 사용자 `state.json`의 boolean만 표시하고 선택한 스크립트만 실행합니다.
실제 설치 여부·버전 점검, 사용자에게 설치/업데이트를 묻는 일, 성공 상태 기록은 개별 스크립트의 책임입니다.
현재 카탈로그에는 OMP 대화형 설치기만 등록되어 있습니다. 플러그인도 같은 entry/state 계약으로 추가합니다.

- [설치·상태 JSON·스크립트 추가·부분/전체 교체](docs/INSTALLATION.md)
- [Release에 업로드할 정확한 파일 목록과 빌드 방법](docs/RELEASE.md)
- 상태·사용자 설정은 배포 경로 밖에 보존하며 배포 기본값은 `defaults/`로 분리합니다.
- 사용자 PC에 Go/Python은 필요 없습니다. 관리 도구 실행 파일을 묶음에 포함합니다.
- GitHub Actions는 `normal`, `normal_develop`, `feature/1/**` push 및 대상 PR에서 실행합니다.
- 최종 **omp-harness-release-files** artifact를 받으면 Release에 올릴 `.exe`, `.run`, `.zip`, `SHA256SUMS`가 있습니다.

Linux 예시:

```sh
sh omp-harness-scripts-0.2.0-linux.run
sh "$HOME/.local/share/omp-harness/manage.sh"
```

Windows에서는 설치 후 시작 메뉴 **Script and plugin manager**를 실행합니다.
설치 상태는 마지막 JSON 기록이며, 관리 도구가 실제 시스템을 다시 검사하지 않습니다.

---

[can1357/oh-my-pi](https://github.com/can1357/oh-my-pi)의 **공식 배포 바이너리**를 설치합니다. 소스나 기능을 변경하지 않습니다. 여기서 샌드박스는 Bun, Node.js, npm, Git 등의 개발 의존성이 없거나 일부만 설치된 OS를 뜻합니다. 격리 환경을 생성하는 스크립트는 아닙니다.

## Linux

```sh
sh install-omp.sh
# 특정 공식 릴리스 설치
sh install-omp.sh v18.8.9
```

- x64/ARM64 Linux, 기본 POSIX 셸과 OS 기본 명령이 필요합니다.
- Debian/Ubuntu, Alpine, Fedora/RHEL 계열, openSUSE, Arch의 패키지 관리자를 지원합니다.
- curl, CA 인증서, 체크섬 및 기본 유틸리티가 없으면 시스템 패키지 관리자로 보충합니다. Alpine은 libstdc++와 libgcc도 설치합니다. 이미 준비된 환경에서는 패키지 설치를 건너뜁니다.
- 패키지 설치에만 root 또는 sudo가 필요합니다. 일반 사용자로 실행하면 OMP는 해당 사용자에게 설치됩니다. 스크립트 전체를 sudo로 실행하면 root 사용자에게 설치됩니다.
- Arch는 부분 업그레이드를 피하기 위해 의존성 보충 시 `pacman -Syu`로 시스템도 업데이트합니다.
- 기본 위치는 `$HOME/.local/bin/omp`입니다. 해당 디렉터리를 PATH에 추가하거나 전체 경로로 실행하세요. `PI_INSTALL_DIR`로 변경할 수 있습니다.

## Windows

```bat
install-omp.bat
rem 특정 공식 릴리스 설치
install-omp.bat v18.8.9
```

- Windows 10/11 또는 Server 2016 이상, x64/ARM64, OS 기본 Windows PowerShell 5.1이 필요합니다. 별도 PowerShell 파일은 필요 없습니다.
- Windows 기본 HTTPS 다운로드와 SHA-256 기능을 사용합니다. Bun, Node.js, Git, winget, WSL, 관리자 권한이 필요하지 않습니다.
- 기본 위치는 `%LOCALAPPDATA%\omp\omp.exe`이며, 공식 설치기가 사용자 PATH에 추가합니다. 설치 후 새 터미널을 여세요. `PI_INSTALL_DIR`로 위치를 변경할 수 있습니다.
- 실행 정책 우회는 해당 PowerShell 프로세스에만 적용합니다. 조직의 실행 제한 정책은 별도 적용될 수 있습니다.

## 설치 동작과 범위

두 스크립트는 `https://omp.sh/install` 또는 `https://omp.sh/install.ps1`을 임시 파일로 내려받고 **binary 모드**로 실행합니다. 공식 `https://build.stencil.so`의 최신 stable 릴리스(또는 지정한 릴리스)를 사용합니다. 공식 설치기가 바이너리 SHA-256을 검증하고, 래퍼가 `omp --version`의 성공 여부를 확인합니다. 재실행하면 공식 바이너리를 다시 설치하며, 실패하면 0이 아닌 종료 코드를 반환합니다. TLS 검증은 끄지 않습니다.

인터넷, 쓰기 가능한 사용자/임시 디렉터리, OS 기본 도구와 작동하는 패키지 저장소는 필요합니다. 사설 프록시 환경에서는 해당 OS/다운로드 도구의 프록시·CA 신뢰 설정을 먼저 제공하세요. 지원하지 않는 OS나 권한 부족은 오류로 안내합니다.

기존 Bun/Node/Git 유무와 관계없이 동일한 공식 바이너리 방식을 사용하므로 개발 런타임을 설치하거나 업그레이드하지 않습니다. 소스 빌드 도구, 프로젝트별 언어 서버, Python 및 모델 API 인증은 설치 범위에 포함되지 않습니다. 설치 후 `omp`를 실행해 모델 제공자 로그인을 진행하세요. Windows 공식 설치기는 기존 Git Bash를 발견하면 OMP의 shellPath를 설정할 수 있습니다.

GitHub Actions는 최소 Debian/Alpine 컨테이너와 Windows에서 설치 및 재설치를 검사합니다. Windows 호스팅 러너는 완전히 빈 OS가 아니며, ARM64 실행 검증은 포함하지 않습니다.

## 대화형 TUI 설치

Linux: `sh install-omp-tui.sh`. Windows: `install-omp-tui.bat` (같은 폴더의 `install-omp-tui.ps1` 필요).
번호/텍스트 입력과 Enter 기본값을 사용하는 터미널 마법사입니다. 프롬프트는 콘솔 인코딩 호환성을 위해 영어입니다. Ctrl+C로 취소합니다.
버전, 기존 설치 교체, 사용자/시스템 범위, 경로, 프록시, PATH, 설정 백업/초기화, 선택 도구, 설치 후 실행을 선택하고 최종 요약을 확인합니다.
Linux는 기존 설치기를 호출하며 의존성 설치 및 Arch 시스템 업데이트를 사전에 안내합니다. 시스템 범위는 root 실행이 필요합니다. 선택 도구 자동 설치는 apt/apk/dnf를 지원합니다. 자동완성은 파일로 저장하며 셸 설정에서 직접 불러옵니다.
Windows는 공식 빌드 서비스에서 SHA-256 검증 후 바이너리를 설치합니다. 시스템 범위에는 관리자 터미널이 필요합니다. 기본 PowerShell로 설치하며 선택 도구에만 winget이 필요합니다. 사설 CA는 Windows 인증서 저장소에서 준비합니다. 셸 설정 유지/내장/Git Bash를 선택할 수 있습니다.
설정 변경 대상은 현재 사용자의 `~/.omp/agent/settings.json`이며 세션 및 인증 데이터는 초기화하지 않습니다. 백업은 같은 디렉터리에 고유 이름으로 보관합니다. 언어 서버는 프로젝트별로 별도 설치하세요.

