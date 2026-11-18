# OMP Harness

## 일반 설치·반복 설치 메뉴·다른 PC 복원 (0.3.0)

[DSH 작업 요약](https://github.com/kwangyeon1/dsh-harness/blob/fb6e0becf1e867e75ca221372dcca0bef5608724/docs/DAILY_SUMMARY_2026-11-17.md)의 흐름을 OMP에 적용했다.
일반 설치와 TUI는 **동일한 개별 공식 설치 스크립트**를 실행한다. OMP는 소스 빌드 대신 공식 바이너리를 사용한다.
사용 중 추가한 설정·확장 소스·플러그인 의존성 기록은 스냅샷으로 보관하고, 다른 PC에서 동일 OMP 버전과 잠금 파일로 복원한다.

- [설치·매개변수·상태 기록·배포 갱신](docs/INSTALLATION.md)
- [환경 보관·Git 이동·별도 복원 TUI·지원 범위](docs/RESTORE.md)
- [Release 파일과 빌드 절차](docs/RELEASE.md)

`.exe`·`.run`은 관리 도구와 스크립트를 배치하며 OMP를 자동 설치하지 않는다. 배포 묶음에는 Linux/Windows x64·ARM64 관리자 실행 파일이 포함되어 사용자 PC에 Go/Python이 필요 없다.
소스 체크아웃은 `python3 packaging/build.py linux` 또는 `windows`로 빌드한 뒤 `dist/bundle`에서 관리 도구를 실행한다.

```sh
# 배포 묶음 안에서
sh manage.sh                       # 선택 → 사용법 → 인자 → 확인 → 실행 → 메뉴 복귀
sh manage.sh help omp
sh manage.sh run omp v18.8.9        # 일반 설치 스크립트 직접 실행
sh manage.sh --project "/absolute/local/omp-harness"

# 사용 중인 OMP를 종료한 뒤 PC A에서 보관
sh manage.sh snapshot omp "$HOME/.local/bin/omp" "$HOME/.omp" "/absolute/saved-environment"
# 파일 검토 후 Git commit·push 및 PC B의 clone은 사용자가 직접 수행
sh restore-tui.sh "/absolute/cloned-environment"
sh restore.sh run "/absolute/cloned-environment" omp "/absolute/new-runtime"
sh "/absolute/new-runtime/start-restored.sh"
```

Windows는 `manage.cmd`, `restore-tui.bat`, `restore.bat`, `start-restored.ps1`을 사용한다.
기존 `install-omp-tui.sh`/`.bat`/`.ps1`은 통합 설치 메뉴의 호환 진입점이다. 별도의 설치 엔진·설정 초기화 마법사는 제거했다.
Git clone/push 기능은 제공하지 않으며 로컬 프로젝트 `--project PATH`만 지원한다.


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

