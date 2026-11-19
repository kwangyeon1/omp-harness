# OMP Harness

## 일반 설치·반복 설치 메뉴·다른 PC 복원 (0.3.1)

[DSH 작업 요약](https://github.com/kwangyeon1/dsh-harness/blob/fb6e0becf1e867e75ca221372dcca0bef5608724/docs/DAILY_SUMMARY_2026-11-17.md)의 흐름을 OMP에 적용했다.
일반 설치와 TUI는 **동일한 개별 공식 설치 스크립트**를 실행한다. OMP는 공식 배포 바이너리를 설치한다.
사용 중 추가한 설정·확장 소스·플러그인 의존성 기록은 스냅샷으로 보관하고, 다른 PC에서 동일 OMP 버전과 잠금 파일로 복원한다.
**0.3.1부터 외부 폴더에 연결한 플러그인과 사용자·이름 있는 프로필의 Marketplace 플러그인도 함께 보관·복원한다.**

- [설치·매개변수·상태 기록·배포 갱신](docs/INSTALLATION.md)
- [환경 보관·Git 이동·별도 복원 TUI·지원 범위](docs/RESTORE.md)
- [Release 파일과 빌드 절차](docs/RELEASE.md)

## 0.3.1 다운로드

Windows 설치 파일, Linux 설치 파일, 공용 ZIP과 체크섬을 한 묶음으로 제공한다. GitHub 로그인과 저장소 접근 권한이 필요하다. 다운로드 후 바깥 ZIP을 풀면 다음 파일이 있다. [설치 파일 묶음 다운로드](https://github.com/kwangyeon1/omp-harness/actions/runs/44430090957/artifacts/14485583772)

| 파일 | 용도 |
| --- | --- |
| `omp-harness-scripts-0.3.1-windows-setup.exe` | Windows 관리 도구·스크립트 설치 |
| `omp-harness-scripts-0.3.1-linux.run` | Linux 관리 도구·스크립트 설치 |
| `omp-harness-bundle-0.3.1.zip` | Linux/Windows x64·ARM64 공용 묶음 |
| `SHA256SUMS` | 세 배포 파일의 SHA-256 |

위 묶음은 구현 커밋 `cc6260c`에서 생성했다. Actions artifact는 보관 기간이 있으며, 확인 시 만료 예정일은 한국 시간 2027-02-17이다. [빌드·검증 결과](https://github.com/kwangyeon1/omp-harness/actions/runs/44430090957)

## 설치 메뉴와 빠른 실행

`.exe`·`.run`은 관리 도구와 스크립트를 배치하며 OMP를 자동 설치하지 않는다. 배포 묶음에는 Linux/Windows x64·ARM64 관리자 실행 파일이 포함되어 사용자 PC에 Go/Python이 필요 없다.
소스 체크아웃은 `python3 packaging/build.py linux` 또는 `windows`로 빌드한 뒤 `dist/bundle`에서 관리 도구를 실행한다.

Linux에서는 `.run`을 실행한 뒤 설치 경로의 관리 메뉴를 연다.

```sh
sh omp-harness-scripts-0.3.1-linux.run
sh "$HOME/.local/share/omp-harness/manage.sh"
```

Windows에서는 `.exe` 실행 후 시작 메뉴의 **Script and plugin manager**를 연다.
공용 ZIP을 풀어 실행할 수도 있다. 아래 관리 명령은 설치 폴더 또는 공용 ZIP을 푼 폴더에서 실행한다.

```sh
sh manage.sh                       # 선택 → 사용법 → 인자 → 확인 → 실행 → 메뉴 복귀
sh manage.sh help omp
sh manage.sh run omp v18.8.9        # 특정 OMP 버전 설치
sh manage.sh --project "/absolute/local/omp-harness"
```

Windows는 `manage.cmd`에 같은 인자를 전달한다. `install-omp-tui.sh`/`.bat`/`.ps1`은 동일한 설치 메뉴로 연결한다.
로컬 설치 스크립트 프로젝트는 `--project PATH`로 선택한다.

## A컴퓨터에서 보관하고 B컴퓨터에서 복원

A와 B에 0.3.1 관리 도구를 준비한다. A의 OMP 및 관련 모듈을 종료한 뒤 새 스냅샷을 만든다.
아래 `OMP_HOME`에 해당하는 경로는 **agent/를 포함하는 설정 루트**이며 기본값은 `~/.omp`다.

Linux의 A에서 실행한다.

```sh
sh manage.sh snapshot omp "$HOME/.local/bin/omp" "$HOME/.omp" "/absolute/saved-environment"
```

Windows의 A에서는 다음 명령을 사용한다.

```bat
manage.cmd snapshot omp "%LOCALAPPDATA%\omp\omp.exe" "%USERPROFILE%\.omp" "C:\saved-environment"
```

생성된 폴더 전체를 B로 옮긴다. Git을 사용할 경우 파일을 검토한 뒤 직접 commit·push하고 B에서 clone한다.
0.3.1에서 추가한 플러그인 소스와 연결 정보까지 포함하려면 A에서 **새 경로에 다시 보관**해야 한다.

Linux의 B에서 복원 메뉴 또는 직접 명령을 실행한다.

```sh
sh restore-tui.sh "/absolute/cloned-environment"
# 직접 복원
sh restore.sh run "/absolute/cloned-environment" omp "/absolute/new-runtime"
# 복원된 환경 확인·실행
sh "/absolute/new-runtime/start-restored.sh" plugin list
sh "/absolute/new-runtime/start-restored.sh"
```

Windows의 B에서는 다음 명령을 사용한다.

```bat
restore-tui.bat "C:\cloned-environment"
rem 직접 복원
restore.bat run "C:\cloned-environment" omp "C:\new-runtime"
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\new-runtime\start-restored.ps1" plugin list
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\new-runtime\start-restored.ps1"
```

복원 대상은 새 경로를 사용한다. Windows에서는 사용자 홈과 같은 드라이브에 둔다.
B는 보관한 릴리스의 OMP 바이너리를 내려받고 의존성을 설치하므로 인터넷이 필요하다.
복원 실행기는 해당 런타임의 설정과 플러그인을 사용한다. 이름 있는 프로필은 실행기에 `--profile NAME`을 전달해 선택한다.

## 플러그인 복원 방식

| A의 설치 방식 | 보관하는 자료 | B에서 준비하는 내용 |
| --- | --- | --- |
| npm·Git 플러그인 | `package.json`, Bun 잠금 파일, 활성화·기능 선택·설정 | 잠금 파일 기준 의존성 설치 |
| `omp plugin link`로 연결한 외부 플러그인 | 실제 소스와 수정 파일, 의존성 기록, 연결 정보 | 복원된 소스에 Linux 링크 또는 Windows 정션 생성 |
| 사용자·이름 있는 프로필의 Marketplace 플러그인 | 설치된 버전의 플러그인 파일, 카탈로그 JSON, 설치 경로·버전·설정·활성화 기록 | 새 경로로 등록 정보 연결 및 런타임 링크 생성 |

외부 연결 플러그인은 `linked-plugins/`에 소스를 보관하고 `plugin-links.json`에 재연결 정보를 기록한다.
Marketplace에서는 설치 기록이 참조하는 플러그인 캐시를 보관한다. B는 보관된 파일을 사용하므로 A의 외부 플러그인 원본 폴더가 없어도 실행 준비가 가능하다.

외부 연결·Marketplace 소스에 의존성이 있고 Bun 잠금 파일이 없으면 A의 OMP에 내장된 Bun으로 **보관 사본에 잠금 파일을 생성**한다.
이때 처음 해석한 의존성 버전은 A에 설치돼 있던 버전과 다를 수 있다. B에서는 생성한 잠금 파일로 설치한다.
npm·Git 플러그인의 기존 의존성 manifest에는 `bun.lock` 또는 `bun.lockb`를 준비한다. Git 의존성에는 Git이 필요하다.

프로젝트 전용 플러그인은 해당 프로젝트와 함께 옮긴다. 로컬 Marketplace의 향후 업데이트에는 원본 소스 경로도 별도로 준비한다.
XDG나 별도 agent 경로로 분산한 자료는 표준 설정 루트에 모아 보관한다.
인증 파일·DB·세션·키와 일반 캐시는 수집에서 제외하며 새 PC에서 다시 로그인한다. 설정 파일에 직접 입력한 비밀값은 Git 반영 전에 검토한다.
추가 경로·의존성·파일 보호 조건은 복원 안내에 정리했다. [상세 복원 안내](docs/RESTORE.md)

## 확인한 동작

- Linux·Windows CI에서 관리자 테스트와 `go vet`, 패키지 설치·재설치·제거 검사 통과.
- A의 원본 경로를 사용할 수 없는 상태에서 외부 연결 소스와 Marketplace 스킬 파일 접근, 비활성화·설정 유지 및 재보관 검사 통과.
- Linux에서 공식 OMP `18.22.2`로 실제 설치·보관·복원 후 `plugin list`, Marketplace 플러그인 재활성화 및 플러그인 의존성 설치 확인.
- ARM64 관리자는 교차 빌드 검증이며 실제 ARM64 실행과 모델 API를 사용하는 플러그인 세션 검증은 남아 있다.

관련 구현과 검사 결과는 이슈 #5에 정리했다. [플러그인 복원 문제 해결 이슈](https://github.com/kwangyeon1/omp-harness/issues/5)

## 개별 공식 설치 스크립트


개별 스크립트는 개발 런타임이 준비되지 않은 OS에서도 공식 OMP 배포 바이너리를 설치합니다. [공식 OMP 저장소](https://github.com/can1357/oh-my-pi)

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

`Build script installers` workflow는 `normal`, `normal_develop`, `feature/**` 브랜치에서 관리자와 배포 패키지를 검사합니다. 공식 OMP 다운로드·재설치는 별도 `Installer smoke tests` workflow에서 최소 Debian/Alpine 컨테이너와 Windows로 검사하며 수동 실행할 수 있습니다.

