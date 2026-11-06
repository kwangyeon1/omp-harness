# 순정 OMP 설치

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
