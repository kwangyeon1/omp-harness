# OMP 환경 보관과 다른 PC에서 복원

DSH의 [2026-11-17 작업 요약](https://github.com/kwangyeon1/dsh-harness/blob/fb6e0becf1e867e75ca221372dcca0bef5608724/docs/DAILY_SUMMARY_2026-11-17.md)을 OMP의 **공식 바이너리·Bun 플러그인 구조**에 맞게 적용했다.
OMP 자체 확장 기능을 재구현하지 않는다. Git commit/push/clone은 사용자가 수행한다.

## PC A: 보관

OMP 및 관련 모듈을 종료한 뒤 실행한다. 소스 체크아웃은 먼저 패키지를 빌드하고 `dist/bundle`에서 실행한다.
`OMP_HOME`은 **agent/를 포함하는 설정 루트**이며 기본값은 `~/.omp`다. 바이너리 경로와 설정 루트를 명시한다.

```sh
sh manage.sh snapshot omp "$HOME/.local/bin/omp" "$HOME/.omp" "/absolute/saved-environment"
sh manage.sh snapshot module extra "/absolute/extra-module" "/absolute/saved-environment"
```

```bat
manage.cmd snapshot omp "%LOCALAPPDATA%\omp\omp.exe" "%USERPROFILE%\.omp" "C:\saved-environment"
manage.cmd snapshot module extra "C:\extra-module" "C:\saved-environment"
```

보관 자료:

```text
saved-environment/
├─ environment.json       종류, OMP 릴리스, 원래 설정 경로, 파일 SHA-256
├─ omp/
│  ├─ home/               agent/, profiles/, plugins/ 등의 실제 사용자 파일
│  └─ home-marker
└─ extra/                 restore.sh / restore.ps1 및 모듈별 자료
```

`omp --version`으로 릴리스를 확인한다. 바이너리를 복사하지 않으므로 새 PC의 OS·CPU에 맞는 **동일 버전**을 내려받는다.
설정·확장·스킬·명령·테마 소스와 플러그인 manifest/Bun 잠금 파일, 활성화·기능 선택·설정 기록을 보관한다.
원본 파일을 수정하지 않으며 같은 스냅샷 ID를 덮어쓰지 않는다. 재보관은 새 폴더를 사용한다.

파일을 검토한 뒤 사용자가 직접 Git에 commit·push하고 새 PC에서 clone한다. Git clone만으로 실행 준비가 끝나지는 않는다.

## PC B: 복원

```sh
sh restore-tui.sh "/absolute/cloned-environment"
# 직접 호출
sh restore.sh list "/absolute/cloned-environment"
sh restore.sh run "/absolute/cloned-environment" omp "/absolute/new-runtime"
sh "/absolute/new-runtime/start-restored.sh" --version
# 프로젝트 디렉터리에서 실행하면 그 프로젝트를 작업 대상으로 사용한다.
sh "/absolute/new-runtime/start-restored.sh"
```

```bat
restore-tui.bat "C:\cloned-environment"
restore.bat run "C:\cloned-environment" omp "C:\new-runtime"
powershell -NoProfile -ExecutionPolicy Bypass -File "C:\new-runtime\start-restored.ps1" --version
```

복원 TUI는 환경 선택 → 항목 → 새 경로 → 확인 → 실행 → 메뉴 복귀 흐름이다. 취소·실패 후에도 다른 항목을 선택할 수 있다.
복원 대상은 **존재하지 않는 경로**여야 하며 스냅샷과 서로 포함할 수 없다. Windows는 사용자 홈과 같은 드라이브를 사용한다.

1. manifest·파일 해시·경로를 검사한다. 텍스트 해시는 Git의 CRLF/LF 변환을 허용하고 내용 변경은 거부한다.
2. 새 경로에 검증된 파일을 배치하고 JSON/YAML의 기존 설정 루트 참조를 새 경로로 바꾼다.
3. 일반 `install-omp.sh`/`.bat`를 보관 릴리스와 `PI_INSTALL_DIR=new-runtime/bin`으로 호출한다. 설치 결과 버전도 일치해야 한다.
4. 의존성이 선언된 각 `package.json` 디렉터리에서 OMP에 내장된 Bun(`BUN_BE_BUN=1`)으로 `install --frozen-lockfile`을 실행한다. 별도 Bun/Node 설치는 필요하지 않다. Git 의존성에는 Git이 필요하다.
5. 실행기와 **모든 단계 성공 후에만** `restored.json`을 생성한다. 실패 파일은 원인 분석을 위해 보존한다. 재시도는 새 경로를 사용한다.

```text
new-runtime/
├─ bin/omp[.exe]
├─ home/                         복원된 설정·확장·플러그인
├─ restore-config-path.json
├─ start-restored.sh / .ps1
└─ restored.json
```

실행기는 `PI_CONFIG_DIR`와 `PI_CODING_AGENT_DIR`를 지정하고 상속된 프로필·XDG 경로를 해제한다.
기본 프로필로 실행하며, 보관한 이름 있는 프로필은 `--profile NAME`으로 선택한다.
프로세스의 현재 작업 디렉터리는 유지한다. 실행기는 생성 위치에 고정되므로 런타임 폴더를 이동하려면 새 경로로 다시 복원한다.
복원은 설치 메뉴의 전역 `state.json`에 성공을 기록하지 않는다. 복원 완료 여부는 해당 런타임의 `restored.json`으로 확인한다.

## 모듈과 지원 범위

다른 모듈은 `restore.sh` 또는 `restore.ps1`을 가진 디렉터리로 보관한다.
복원 시 해당 OS의 스크립트를 실행하며 작업 디렉터리와 첫 인자는 새 모듈 경로다.
스크립트는 모듈별 도구 준비·의존성 설치·빌드를 담당하고 실패 시 0이 아닌 종료 코드를 반환해야 한다.
신뢰하는 보관 자료와 복원 스크립트만 실행한다. 해시는 무결성 검사이며 서명은 아니다.

- 공식 바이너리 설치 환경을 지원한다. OMP 소스 수정본·npm 런타임 자체의 복원은 지원하지 않는다.
- 심볼릭 링크는 자동으로 따라가지 않으며 보관을 거부한다. 실제 소스를 별도 경로에 준비한다.
- `node_modules`를 수집하지 않는다. 사용자 제작 확장·플러그인은 설정 루트 안에 실제 소스로 두고, 의존성이 있으면 해당 manifest 옆에 `bun.lock` 또는 `bun.lockb`를 준비한다. 외부 `file:`/`link:` 경로는 별도로 준비해야 한다.
- `omp plugin link`만으로 등록한 외부 소스와 marketplace 캐시/링크는 일반 복원 대상이 아니다. 해당 항목을 발견하면 보관을 거부하며 모듈별 복원 어댑터를 요구한다. npm/Git 플러그인은 manifest와 잠금 파일로 복원하고 활성화 상태를 유지한다.
- `PI_CODING_AGENT_DIR`로 분리한 agent 폴더 또는 XDG로 분산한 데이터는 자동 수집하지 않는다. 먼저 별도 작업 폴더에 agent/·plugins/ 등 표준 루트 구조로 모으고 경로 참조도 맞추거나 모듈 어댑터를 사용한다.
- 인증 파일(`auth.json*`, credentials), `.env*`, `.npmrc`, `.netrc`, DB/SQLite, 키 파일, node_modules, 세션·로그·캐시·런타임 파일은 제외한다. DB에만 있는 설정도 복원하지 않는다. 인증은 새 PC에서 다시 한다.
- 설정 JSON/YAML 등에 직접 쓴 비밀값은 자동 판별하지 않는다. Git 반영 전 파일 검토가 필요하다.
- 외부 프로젝트·서비스·언어 서버·OS 전용 도구는 별도로 준비한다. 다른 OS의 바이너리 제공 여부와 플러그인 호환성은 해당 릴리스/플러그인에 따른다.
- 실제 다운로드에는 공식 서비스가 해당 릴리스를 계속 제공해야 한다. 오프라인 복원·자동 동기화·자동 Git 작업은 제공하지 않는다.
