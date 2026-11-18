# omp-harness 작업 인계

작성일: 2026-11-13 UTC. 이 문서는 `feature/1/normal_develop_설치방식수정`의 0.2.0 구현 상태를 설명한다.
과거 0.1.0 설명보다 이 문서와 [설치 계약](docs/INSTALLATION.md), [Release 절차](docs/RELEASE.md)를 우선한다.

## 목표와 사용자 요구

- `.exe`/`.run`은 배포 묶음을 배치한다. OMP 설치나 플러그인 연결을 자동으로 이어서 실행하지 않는다.
- 관리 도구는 JSON의 설치됨/안됨만 읽는다. 대상 프로그램, 플러그인, 레지스트리, 설치 경로, 버전을 탐지하지 않는다.
- 사용자가 선택한 개별 스크립트가 필요하면 직접 점검하고 설치/업데이트 여부를 묻는다.
- 개별 스크립트만 성공 지점에서 상태를 기록한다. 취소/실패 또는 단순 종료 코드 0으로 상위 관리자가 상태를 추정하지 않는다.
- 사용자 상태/설정과 새 배포 기본값을 분리한다. 묶음을 전체 또는 구성요소별로 교체할 수 있어야 한다.
- Windows와 Linux를 지원한다. Release에 업로드할 실제 파일 목록을 설명한다.
- 이번 요청은 위 브랜치 수정·커밋·푸시 및 이슈 #1 본문 갱신이다. 병합, 태그, Release 공개 요청은 없다.

## 확인한 구현과 주요 결정

관리자는 Go 표준 라이브러리로 작성하고 빌드 시 Windows/Linux amd64/arm64 실행 파일을 생성한다.
배포받은 사용자는 Go/Python을 설치할 필요가 없다. 셸/PowerShell 개별 스크립트는 기존 방식으로 실행한다.
이 선택은 양 플랫폼에서 JSON 쓰기와 묶음 교체를 동일한 계약으로 제공하기 위한 구현 결정이다.

- `catalog.json`: 항목 ID, 이름, kind, 플랫폼별 entry. 현재 실제 등록 항목은 OMP 하나다.
- `manager/main.go`: JSON 표시, 선택 실행, 명시적 상태 기록. 상태는 boolean map이며 잘못된 JSON은 보존하고 오류를 낸다.
- `manager/bundle.go`: 파일 소유권/해시 검사, 전체·부분 교체, 이전 소유 파일 정리, 일반 쓰기 실패 시 복구 시도.
- `manage.sh`, `manage.cmd`: 플랫폼 실행 파일 선택.
- `install-omp-tui.sh`, `.ps1`: 최종 설치 동의 기본값 no, 성공 후에만 상태 기록.
- `defaults/settings.json`: 현재 `{}`. 관리자는 기본값/사용자 설정의 경로만 전달한다. 개별 스크립트가 사용자 값 우선 정책을 구현한다.
- `packaging/build.py`, `components.json`: 구성요소 파일 소유권 선언, bundle.json 생성, 네이티브 관리자와 설치 패키지 빌드.
- `packaging/windows.iss`, `windows-launch.bat`: Inno Setup 사용자 범위 설치, 관리 메뉴/OMP 선택 실행.
- `packaging/linux-install.sh`, `linux-uninstall.sh`: 묶음 적용/제거. 소유하지 않은 파일과 외부 사용자 상태를 유지한다.
- `manager/main_test.go`, `packaging/test_linux.py`, `test_windows.ps1`: 계약/설치 패키지 검사.
- `.github/workflows/package.yml`: 두 OS 빌드·검사, 최종 Release 업로드용 artifact 취합.
- `README.md`, `docs/INSTALLATION.md`, `docs/RELEASE.md`, 이 문서: 사용/개발/배포 설명.

사용자 상태 기본 경로는 Linux `~/.local/state/omp-harness/state.json`(XDG_STATE_HOME 지원),
Windows `%LOCALAPPDATA%\omp-harness\state.json`이다. `OMP_HARNESS_STATE_FILE`로 변경할 수 있다.
사용자 설정은 기본 상태 디렉터리의 `settings.json`이다. OMP 자체 설정/인증/세션은 기존 `~/.omp`에 있다.

## 브랜치와 커밋 확인

저장소: <https://github.com/kwangyeon1/omp-harness> (비공개).
작업 브랜치: `feature/1/normal_develop_설치방식수정`.
작업 시작 시 원격 HEAD: `3bf267f491a0c8169d1c0f20a605b3ccd6271e6f`.
패키지 버전: `0.2.0`. 이 문서 자신의 커밋 SHA는 고정하지 않는다.

```sh
git fetch origin
git log -3 --oneline 'origin/feature/1/normal_develop_설치방식수정'
```

최종 커밋, CI 결과와 다운로드 링크는 [이슈 #1](https://github.com/kwangyeon1/omp-harness/issues/1)에 갱신한다.
새 세션에서는 이슈와 원격 HEAD를 재확인해야 한다.

## 검증한 사실과 한계

로컬 Linux에서 Go 계약 검사 6개, `go vet`, Linux 패키지 빌드 및 Python smoke 검사 5개가 통과했다.
검사는 JSON만으로 상태 표시, 취소/실패 시 기록 안 함, 실제 leaf의 명시적 성공 기록,
공백 경로 설치/재설치/제거, 부분/전체 교체, 폐기 파일 정리, 사용자 파일/상태 보존,
손상된 해시·경로·동시 작업 잠금·사용자 파일 충돌 거부를 포함한다.
Windows 설치와 Windows 명령 경로 검증은 GitHub Actions의 Windows runner에서 수행한다.
이 문서 작성 시점의 로컬 테스트 결과와 이후 CI 결과를 혼동하지 말 것.
실제 OMP 원격 다운로드, 실제 ARM64 실행, 수동 Windows GUI 검사는 이번 계약 검사의 범위가 아니다.

## 미구현/미확인 사항

- 자동 신버전 검색/다운로드 클라이언트 및 자동 Release 공개는 없다. Release ZIP을 직접 받아 적용한다.
- OMP leaf는 latest/특정 버전 재설치를 제공하지만 설치 버전과 원격 최신 버전 비교 알림은 구현하지 않았다.
- 실제 플러그인 패키지는 아직 지정되지 않았다. plugin kind도 동일한 JSON/스크립트 계약을 사용한다.
- Windows `.exe` 재설치는 파일을 갱신하지만 이전 manifest의 폐기 파일을 정리하지 않는다. 정확한 정리는 새 ZIP의 `bundle apply`로 수행한다.
- 일반 파일 쓰기 오류 복구는 시도하지만 전원 단절/강제 종료 복구 저널은 없다. 중단된 `.lock`은 다른 실행이 없는지 확인 후 제거한다.
- 코드 서명/묶음 서명은 없다. SHA-256은 무결성 확인이며 서명이 아니다.
- 비공개 Release의 무인 다운로드 인증 정책은 정해지지 않았다.
- 전체 적용은 기존 버전/신규 설치에 가능하다. 부분 적용은 manifest가 있는 기존 설치에만 허용한다.

## 이어서 작업하는 방법

1. 원격 브랜치, 이슈 #1, 최신 Actions 결과를 확인한다. 실패 중인 CI가 있으면 먼저 원인을 읽는다.
2. 새 항목은 scripts/ 또는 plugins/에 leaf를 두고 catalog.json과 packaging/components.json에 등록한다.
3. leaf 안에서만 점검/사용자 질문/성공 상태 기록을 수행한다. 관리 도구에 제품 탐지 코드를 추가하지 않는다.
4. 필요 변경 후 `go test ./manager`, `go vet ./manager`, 빌드 및 해당 OS smoke 검사를 실행한다.
5. Release 요청 시 성공한 빌드의 `omp-harness-release-files` artifact에서 `.exe`, `.run`, `.zip`, `SHA256SUMS`를 받아 동일 Release에 첨부한다. 정확한 파일명과 절차는 docs/RELEASE.md 참조.

빌드: Python 3.10+, Go 1.24.9, Windows Inno Setup 6. `OMP_GO`로 Go 경로를 지정할 수 있다.

```sh
go test ./manager
python3 packaging/build.py linux
python3 packaging/test_linux.py
```

이번 환경의 `/workspace/omp-harness`는 원격 브랜치 체크아웃이 아닌 빈 Git 저장소였다.
`/workspace/work/install-redesign`은 작업용 소스와 산출물이며 Git 체크아웃이 아니다.
원격 GitHub 커넥터로 기존 tree 기반 commit을 만들고 `expected_sha`로 ref를 갱신한다.
새 세션은 환경이 같다고 가정하지 말고 Git 상태·AGENTS.md·인증부터 확인한다.
사용자 변경을 강제 덮어쓰지 않는다. 생성 파일은 Git에 커밋하지 않고 Actions 다운로드 링크로 제공한다.
