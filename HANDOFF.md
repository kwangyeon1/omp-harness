# omp-harness 작업 인계

작성일: 2026-11-06 (Asia/Seoul). 이 문서는 Codex의 다음 세션이 대화 기록 없이 작업을 이어가기 위한 문서다.
본문의 상태는 아래 구현 커밋을 기준으로 확인했다. 문서 이후 변경 사항은 반드시 원격에서 다시 확인한다.

## 1. 현재 목표와 완료 상태

사용자의 장기 관심사는 GitHub Releases에 설치 파일/바이너리를 배포하고, 설치된 프로그램이 이를 통해 업데이트되는 구조다.
이번에 실제로 요청받아 구현한 범위는 **기존 설치 스크립트를 담은 Windows 및 Linux 설치 패키지 제작**이다.
이 구현은 원격 `normal` 브랜치에 반영됐고, 두 OS의 CI 빌드와 설치·재설치·제거 검사가 성공했다.

최신 사용자 요청은 이 `HANDOFF.md` 작성이다. 자동 업데이트 구현, 새 Release 공개, 기본 브랜치 병합은 이번 문서 작성 요청에 포함되지 않는다.

다음 네 항목을 혼동하지 말 것:

1. **스크립트 패키지 설치**: `.exe` / `.run`이 기존 스크립트를 사용자 PC에 배치한다. 이 단계는 네트워크를 사용하지 않는다.
2. **OMP 설치**: 사용자가 설치된 스크립트를 실행하면 공식 서버에서 OMP를 다운로드한다. 인터넷이 필요하다.
3. **스크립트 패키지 갱신**: 새 설치 패키지를 같은 위치에 다시 실행하는 수동 방식이다.
4. **Releases 자동 업데이트**: 새 버전 검색·다운로드·검증·자동 적용 클라이언트는 아직 없다.

## 2. 사용자 요구사항과 제약

### 대화에서 명시된 요구

- 사용자는 기존 스크립트를 **옵션으로 제공하는 구성요소**라고 설명했다. 프로젝트의 최종 형태를 현재 스크립트에만 한정하지 말 것.
- 향후 설치 스크립트/설치 파일을 GitHub Releases로 올릴 수 있도록, 바이너리 생성 및 업데이트 준비 과정을 알고 싶어 했다.
- 원격 `omp-normal` 브랜치 생성을 요청했고, 이어서 원격 `normal` 브랜치 생성을 요청했다.
- 설치 패키지 구현은 **`normal` 브랜치에서** 진행하도록 요청했다.
- Windows 구현 진행 중 **Linux 버전도 함께 만들어야 한다**고 추가했다.
- 최신 요청은 목표, 요구/제약, 결정/이유, 완료 작업, 변경 파일, 브랜치/커밋, 테스트, 미해결 사항, 다음 작업을 사실과 추측으로 구분한 인계 문서 작성이다.
- 대화와 설명은 한국어로 진행했다.

### 구현에서 선택한 동작 (사용자의 추가 확정 요구와 구분)

- 설치 파일이 OMP 설치를 즉시 실행하지 않고 스크립트를 배치한 뒤 사용자가 선택 실행하도록 했다.
- 패키지 제거는 패키지 소유 파일만 대상으로 하며, 별도로 설치한 OMP와 설정/인증/세션은 유지하도록 설계했다.
- 사용자 범위 설치를 기본으로 하고 스크립트 패키지 설치 자체에는 관리자 권한을 요구하지 않는다.
- 시스템 의존성 설치나 시스템 범위 OMP 설치에 필요한 권한은 기존 스크립트의 동작을 따른다.
- 코드/빌드 구성을 `normal`에 직접 커밋했다. PR이나 `master` 병합은 하지 않았다.
- Release 공개나 태그 생성은 하지 않았다. 장기 업데이트 구조는 설명했지만 구현하지 않았다.

## 3. 저장소, 브랜치, 커밋

저장소: <https://github.com/kwangyeon1/omp-harness> (비공개)

| 항목 | 문서 작성 직전 원격에서 확인한 값 |
| --- | --- |
| 기본 브랜치 | `master` |
| 작업 브랜치 | `normal` |
| 구현 커밋 / 문서 추가 전 normal HEAD | `2ed0c62fd3274ca7e3a841c2703d5f985a81a9b7` |
| 구현 커밋 메시지 | `Add Windows and Linux installers for optional OMP scripts` |
| 구현 커밋의 tree SHA | `6fec18cb1515815994b78698b96085a3f7eb695a` |
| master HEAD | `61c6b0e01e745a6180b503ce39afad2e5b8880af` |
| 패키지 버전 | `0.1.0` (`packaging/VERSION`) |

이 문서 자체는 위 구현 커밋 이후의 문서 커밋으로 추가한다. 자기 자신의 커밋 SHA를 본문에 고정할 수 없으므로 다음 명령으로 최신 상태와 문서 커밋을 확인한다.

```sh
git fetch origin
git log -1 --format='%H %s' origin/normal
git log -1 --format='%H %s' origin/normal -- HANDOFF.md
```

`omp-normal`은 이전 생성 요청 당시 API 성공 응답을 받았다. 하지만 **이번 재확인에서 해당 원격 ref 조회는 HTTP 404**였다.
삭제 여부/삭제 주체/원인은 확인하지 않았다. 필요한 경우 다시 조회하되, 인계 과정에서 이 브랜치를 재생성하거나 현재 작업 브랜치를 바꾸지 말 것.

## 4. 주요 결정과 이유

| 결정 | 이유 및 현재 동작 |
| --- | --- |
| Windows는 Inno Setup 6 | 기존 `.bat`/`.ps1`을 포함하는 실제 `.exe` 설치 파일과 제거 프로그램, 시작 메뉴 바로가기를 제공할 수 있다. |
| Linux는 셸 헤더 + tar.gz payload의 단일 `.run` | 기존 POSIX 셸 스크립트를 독립 파일로 전달하며 별도 앱 소스 컴파일이나 사용자 PC의 Python 설치를 요구하지 않는다. |
| 빌드 도구는 Python 표준 라이브러리 | Linux payload 생성, 버전 검증, 해시 계산을 외부 Python 패키지 없이 처리한다. Windows 빌드에는 별도 Inno Setup이 필요하다. |
| 고정 Windows `AppId` | 다음 버전 설치 시 동일한 스크립트 패키지로 인식하도록 `omp-harness-script-installer`를 사용한다. |
| 패키지 버전과 OMP 버전 분리 | 설치기만 변경되는 경우도 있으므로 패키지 버전을 OMP 릴리스와 동일시하지 않는다. |
| 우선 Actions artifact로 제공 | 두 OS에서 실제 빌드 및 검사를 완료한 파일을 받을 수 있다. Releases 배포 자동화는 별도 후속 작업이다. |

이 저장소에 독자적인 `omp-harness.exe` 앱 소스는 추가하지 않았다. Windows `.exe`는 **Inno Setup이 만든 설치 프로그램**이다.
과거 설명에 등장한 Bun/Rust/Go 빌드, `updater.exe`, `latest.json`, 서명 키, 버전별 업데이트 zip 등은 설계 예시였다. 현재 구현 파일로 취급하지 말 것.

## 5. 완료한 작업과 변경 파일

구현 커밋에서 변경된 파일은 아래 11개다.

| 파일 | 역할 |
| --- | --- |
| `.github/workflows/package.yml` | `normal` push/대상 PR 및 수동 이벤트용 Windows/Linux 빌드, 검사, artifact 업로드 |
| `.gitignore` | 기존 `/work/` 유지, `dist/`, `__pycache__/` 제외 추가 |
| `README.md` | 패키지 다운로드/설치/빌드/검사/제거/Release 수동 배포 설명 추가 |
| `packaging/VERSION` | 패키지 버전 `0.1.0`; 현재 숫자 3개 형식만 허용 |
| `packaging/build.py` | `linux`/`windows` 빌드 진입점, 설치 파일 및 `.sha256` 생성 |
| `packaging/windows.iss` | 사용자 범위 Windows 설치 파일, 시작 메뉴 및 제거 프로그램 정의 |
| `packaging/windows-launch.bat` | 일반/TUI 설치 스크립트 선택 실행, 종료 코드 표시, 콘솔 유지 |
| `packaging/linux-install.sh` | `.run` 헤더 템플릿, 경로/플랫폼 검사, payload 추출 및 파일 배치 |
| `packaging/linux-uninstall.sh` | 패키지 마커 확인 후 패키지 소유 파일 제거, 사용자 파일 보존 |
| `packaging/test_linux.py` | 생성된 Linux 설치 파일의 설치/재설치/제거 및 오류 처리 검사 |
| `packaging/test_windows.ps1` | Windows silent 설치/재설치/제거, payload 및 해시 검사 |

이번 인계 작업은 루트의 `HANDOFF.md`만 추가한다.

기존 `install-omp.sh`, `install-omp.bat`, `install-omp-tui.sh`, `install-omp-tui.bat`, `install-omp-tui.ps1`의 원격 내용은 구현 커밋에서 변경하지 않았다.
기존 `.github/workflows/install.yml`도 변경하지 않았다. 이 기존 workflow는 `master` push 및 수동 실행 시 OMP 실제 설치/재설치를 검사하는 별도 workflow다.

### 설치 후 구성

Windows 기본 경로: `%LOCALAPPDATA%\Programs\omp-harness`

```text
install-omp.bat
install-omp-tui.bat
install-omp-tui.ps1
launch-installer.bat
VERSION
README.md
unins000.exe 등 Inno Setup 제거 파일
```

시작 메뉴 `OMP Harness Scripts`에 일반 설치/업데이트, 대화형 설치, 스크립트 패키지 제거 바로가기를 만든다.
Windows 10 이상, x64/ARM64 허용 설정이다. Windows 실제 CI 검사는 x64에서만 수행했다.

Linux 기본 경로: `$HOME/.local/share/omp-harness` (`--prefix /absolute/path` 지원)

```text
install-omp.sh
install-omp-tui.sh
uninstall.sh
VERSION
README.md
.omp-harness-package
```

Linux 설치기는 Linux x64/ARM64를 허용하며, 마커 없는 비어 있지 않은 디렉터리와 최종 경로가 심볼릭 링크인 디렉터리는 거부한다.
추가 파일이 있으면 제거 시 그 파일과 디렉터리를 보존한다. 소유 파일 교체는 순차 복사이며 원자적 업데이트/롤백은 구현하지 않았다.

## 6. 빌드와 사용 재현

빌드 환경: Python 3.10 이상. CI는 Python 3.12를 사용한다. Windows에는 Inno Setup 6이 추가로 필요하다.
Linux 패키지 실행 환경: POSIX 셸, gzip 지원 tar, tail, mktemp, uname 및 기본 파일 유틸리티. 사용자 PC에 Python은 필요 없다.

저장소 루트에서 실행:

```sh
# Linux 패키지 빌드 및 검사
python3 packaging/build.py linux
python3 packaging/test_linux.py
sh -n packaging/linux-install.sh
sh -n packaging/linux-uninstall.sh

# 생성 파일의 SHA-256 확인
(cd dist && sha256sum -c omp-harness-scripts-0.1.0-linux.run.sha256)
```

```powershell
# Windows + Inno Setup 6에서 빌드 및 검사
python packaging/build.py windows
powershell -NoProfile -ExecutionPolicy Bypass -File packaging/test_windows.ps1
Get-FileHash dist/omp-harness-scripts-0.1.0-windows-setup.exe -Algorithm SHA256
```

예상 빌드 결과:

```text
dist/omp-harness-scripts-0.1.0-linux.run
dist/omp-harness-scripts-0.1.0-linux.run.sha256
dist/omp-harness-scripts-0.1.0-windows-setup.exe
dist/omp-harness-scripts-0.1.0-windows-setup.exe.sha256
```

Linux 사용자 실행 예:

```sh
sh omp-harness-scripts-0.1.0-linux.run
sh "$HOME/.local/share/omp-harness/install-omp-tui.sh"
# 스크립트 패키지만 제거
sh "$HOME/.local/share/omp-harness/uninstall.sh"
```

버전을 올릴 때는 `packaging/VERSION`을 변경하고 README의 버전 예시도 갱신한다.
Linux 헤더의 `@PAYLOAD_LINE@`은 빌더가 계산해 치환하므로 손으로 payload 시작 행을 고정하지 않는다.
Windows `AppId`는 동일 패키지의 업그레이드 인식을 위해 유지한다.

## 7. 테스트 결과와 빌드 산출물 — 확인한 사실

구현 커밋의 CI: <https://github.com/kwangyeon1/omp-harness/actions/runs/42181346209>

- 전체 run: `completed / success` (이번 인계 작성 시 API 재확인).
- `package (windows-latest, windows)`: 성공.
- `package (ubuntu-latest, linux)`: 성공.
- 양쪽 artifact 업로드: 성공.
- 이전 구현 세션에서 로컬 Linux 빌드, 4개 unittest, 두 셸 템플릿의 `sh -n`이 성공했다. 문서만 추가하는 인계 작업에서는 이 실행을 반복하지 않았다.

| 검사 | 확인된 범위 |
| --- | --- |
| Linux 4개 unittest | 공백 포함 경로 설치/재설치/제거, 원본 payload와 버전 일치, 추가 사용자 파일 보존, 마커 없는 기존 파일 보호, 도움말/잘못된 인자 처리, 산출물 SHA-256 일치 |
| Windows PowerShell 검사 | silent 설치 2회, 스크립트/README 해시와 버전 일치, launcher 존재, silent 제거, 추가 사용자 파일 보존, 설치 파일 SHA-256 일치 |
| 빌드 | Inno Setup으로 Windows `.exe` 생성 및 Linux `.run` 생성 성공 |

이 CI는 **스크립트 패키지 검사**다. 설치 후 스크립트를 실행해 공식 OMP 다운로드까지 수행하는 통합 검사와는 다르다.
기존 `install.yml`의 새 실행 결과나 실제 사용자 GUI 조작, ARM64 실행까지 성공했다고 주장하지 말 것.

다운로드 링크 (비공개 저장소 접근 권한 및 GitHub 로그인 필요):

| OS | Artifact |
| --- | --- |
| Windows | <https://github.com/kwangyeon1/omp-harness/actions/runs/42181346209/artifacts/13484949866> |
| Linux | <https://github.com/kwangyeon1/omp-harness/actions/runs/42181346209/artifacts/13485800440> |

각 artifact ZIP 안에 설치 파일과 `.sha256` 파일이 있다.
재확인 당시 두 artifact는 `expired: false`, 만료 예정은 **2027-02-04 16:30:44 (Asia/Seoul)**이었다. 보관 정책이나 이후 삭제로 다운로드 가능 상태가 바뀔 수 있다.

## 8. 미해결 사항과 검증 한계

### 코드/API로 확인한 미구현 사항

- GitHub Releases 목록은 이번 확인 시 비어 있었다. 설치 파일은 Actions artifact에만 있다.
- 자동 Release 생성/업로드 workflow는 없다. README에는 artifact를 받아 수동 첨부하는 절차만 있다.
- 새 버전 확인, 자동 다운로드/검증/교체/재시작 클라이언트가 없다.
- Windows 코드 서명 및 업데이트 패키지 서명은 구성하지 않았다. `.sha256`은 파일 해시이며 배포자 서명을 대신하지 않는다.
- OMP 바이너리를 설치 패키지에 포함하지 않았다. 오프라인 상태에서 OMP까지 설치되는 패키지는 아니다.
- 독립 앱의 언어/프레임워크, 업데이트 프레임워크, 공개/비공개 배포 인증 방식은 결정되지 않았다.
- `.deb`, `.rpm`, macOS 패키지는 구현하지 않았다. 현재 요구받은 패키지는 Windows `.exe`와 Linux `.run`이다.
- `package.yml`은 `normal` push 또는 `normal` 대상 PR에서 동작한다. `install.yml`은 기본적으로 `master` push에서만 동작한다.

### 추측 또는 후속 검토 대상 — 결함 확정이 아님

- 장기 목표를 이어간다면 Release 배포 자동화가 다음 구현 후보지만, 사용자가 다음으로 그것을 하라고 확정한 상태는 아니다.
- ARM64 환경에서의 실제 실행, 여러 Linux 배포판의 패키지 설치 동작, 시작 메뉴에서의 수동 실행은 별도 검증이 필요하다.
- Linux 설치 중 디스크 부족/중단 등에 대한 트랜잭션 복구는 검사하지 않았다. 현재 코드는 순차 복사라 완전한 롤백을 보장하지 않는다.
- 같은 소스를 반복 빌드했을 때 바이트 단위 재현성은 검사하지 않았다. 릴리스 해시는 실제 배포할 산출물에서 계산해야 한다.
- 비공개 저장소에서 무인 자동 업데이트를 제공하려면 인증 설계가 필요하다. 배포자 토큰을 클라이언트에 내장하는 방식은 사용하지 말 것.

## 9. 새 Codex 세션의 작업 환경 주의사항

이전 작업은 GitHub 커넥터 API로 원격 tree/commit을 만든 뒤 `normal` ref를 갱신했다. 로컬 Git push로 반영한 것이 아니다.

인계 시 실제 로컬 상태:

- `/workspace/omp-harness`에는 `.git`만 있었다.
- `git status --short --branch` 결과는 `## No commits yet on work`였다. 이 로컬 브랜치는 원격 `normal`의 체크아웃이 아니다.
- origin은 `https://github.com/kwangyeon1/omp-harness.git`이었다.
- `/workspace/work/normal-packaging`에는 작업용 소스와 로컬 Linux 산출물이 있다. 이것은 Git 체크아웃이 아니며 `.github/workflows/install.yml` 등 원격 전체를 포함하지 않는다. 원격과 완전히 동일하다고 가정하지 말 것.
- GitHub 커넥터의 읽기/쓰기 동작은 성공했다. 셸용 GitHub 인증이나 새 세션의 커넥터 연결이 동일하게 제공될지는 미확인이다.

새 환경에서는 먼저 cwd, `AGENTS.md`, Git 상태와 원격 HEAD를 확인한다. 빈 로컬 `work` 브랜치를 그대로 push하거나 scratch 디렉터리를 정식 저장소로 취급하지 않는다.
기존 파일을 덮어쓰지 않는 새 경로에 인증된 clone을 만드는 방법이 안전하다. 예시:

```sh
# 현재 디렉터리와 기존 파일을 먼저 확인하고, 아래 대상 경로가 없을 때 실행
mkdir -p /workspace/work
git clone --branch normal --single-branch \
  https://github.com/kwangyeon1/omp-harness.git \
  /workspace/work/omp-harness-normal
cd /workspace/work/omp-harness-normal
git status --short --branch
git log -3 --oneline
```

CLI 인증이 없으면 GitHub 커넥터를 사용한다. 도구 이름/스키마는 세션마다 검색해서 확인한다.
이번에 사용한 기능은 `github_fetch`, `github_fetch_file`, `github_create_tree`, `github_create_commit`, `github_update_ref`다.
수정 시 최신 HEAD와 commit의 tree SHA를 읽고, 그 tree 기반으로 파일을 추가한 뒤 기존 HEAD를 parent로 커밋한다.
ref 갱신에는 `expected_sha`를 전달해 동시 변경을 감지하고, 충돌하면 최신 변경을 재확인한다. 사용자 변경을 강제 덮어쓰지 않는다.

## 10. 다음 작업 순서

1. **원격 재동기화**: `normal` 최신 HEAD와 이 문서 이후 변경, 현재 CI 상태를 확인한다.
2. **요청 범위 확인**: 현재 설치 패키지 구현은 완료됐다. 새 사용자 요청이 없다면 Release 공개나 자동 업데이트 구현을 임의로 시작하지 않는다.
3. **배포를 요청받으면**: 배포 버전과 대상 커밋을 고정하고 성공한 Windows/Linux artifact 및 해시를 확보한다. 공개된 같은 버전 파일을 덮어쓰기보다 새 버전을 발행한다.
4. **Release 자동화를 요청받으면**: 태그/버전 일치 검증, 두 OS 빌드·검사 완료 후 업로드, Release 초안/공개 시점, `contents: write` 및 서명 자격 증명을 설계한다. 현재 읽기 전용 권한 workflow와 구분한다.
5. **자동 업데이트를 요청받으면**: 스크립트 패키지와 OMP 중 무엇을 업데이트할지, 앱/업데이트 도구의 형태, 비공개 자산 인증, 버전 메타데이터, 서명 검증, Windows 종료 후 교체, 롤백 및 사용자 데이터 보존을 정한다.
6. **업데이트 구현 검증**: 구버전 설치 → 신버전 배포 → 감지/적용 → 버전·설정 보존 확인과 다운로드/검증 실패 시 기존 버전 유지까지 두 버전으로 검사한다.

## 11. 사실 확인에 사용한 근거

- 원격 normal: <https://github.com/kwangyeon1/omp-harness/tree/normal>
- 구현 커밋: <https://github.com/kwangyeon1/omp-harness/commit/2ed0c62fd3274ca7e3a841c2703d5f985a81a9b7>
- README: <https://github.com/kwangyeon1/omp-harness/blob/normal/README.md>
- 빌드 workflow: <https://github.com/kwangyeon1/omp-harness/blob/normal/.github/workflows/package.yml>
- CI 실행: <https://github.com/kwangyeon1/omp-harness/actions/runs/42181346209>
- GitHub API로 ref, 커밋 변경 파일, 소스 파일, CI run/jobs/artifacts, Releases 목록을 직접 조회했다.
- 위 로컬 Git 상태는 인계 작성 세션의 셸 출력에서 확인했다. 경로의 존재와 상태는 새 세션에서 달라질 수 있다.
