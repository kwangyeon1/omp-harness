# JSON 기반 스크립트·플러그인 관리

## 책임 경계

관리 도구는 `catalog.json`과 사용자 `state.json`만 읽어 목록을 표시한다.
대상 프로그램/플러그인의 실행 파일, 설치 디렉터리, 레지스트리, 프로세스 또는 버전을 검사하지 않는다.
네트워크로 신버전을 조회하지 않는다. 사용자가 고른 항목의 entry만 실행한다.
`kind: script`와 `kind: plugin`은 동일한 실행 계약을 사용한다.
현재 실제 등록 항목은 OMP 하나다. 배포할 플러그인이 정해지면 아래 계약으로 추가한다.

각 항목의 스크립트가 필요할 때 자체 점검·설치/업데이트 질문·설정 변경을 수행한다.
OMP는 기존 대화형 설치기를 사용하며, 교체/최종 설치를 사용자에게 묻는다.
현재 OMP 스크립트는 최신 버전과 비교하는 업데이트 알림을 구현하지 않았고, 사용자가 latest/특정 버전을 골라 재설치한다.
상위 도구가 종료 코드 0만 보고 설치 상태를 바꾸는 로직은 없다. 취소도 정상 종료일 수 있기 때문이다.

## 사용자 실행

패키지 설치는 파일 배치만 수행한다. 설치 후 자동으로 OMP, 플러그인 또는 다른 배포 스크립트를 실행하지 않는다.

Linux:

```sh
sh omp-harness-scripts-0.2.0-linux.run
sh "$HOME/.local/share/omp-harness/manage.sh"         # 선택 메뉴
sh "$HOME/.local/share/omp-harness/manage.sh" list    # JSON 상태 표시
sh "$HOME/.local/share/omp-harness/manage.sh" run omp # 선택한 스크립트만 실행
```

Windows: `.exe` 설치 후 시작 메뉴의 **Script and plugin manager**를 사용한다.
관리 도구가 설치된 디렉터리에서 `manage.cmd list`, `manage.cmd run omp`도 가능하다.
개별 OMP 대화형 스크립트를 직접 실행해도 패키지에 포함된 관리 도구로 상태를 기록한다.
기존 비대화형 `install-omp.sh`/`.bat`는 하위 설치 수단으로 보존했으며 메뉴에 등록하지 않는다.

## 목록과 상태

배포 파일 `catalog.json`:

```json
{
  "items": [
    {
      "id": "plugin-a",
      "name": "Plugin A",
      "kind": "plugin",
      "entry": {"linux": "plugins/plugin-a/install.sh", "windows": "plugins/plugin-a/install.cmd"}
    }
  ]
}
```

ID는 구성요소를 식별하는 고정 값이다. 기본값은 미설치로 처리하며, 상태를 억지로 생성하거나 실제 설치를 추측하지 않는다.

사용자 `state.json`:

```json
{"omp": true, "plugin-a": false}
```

- Linux: `$XDG_STATE_HOME/omp-harness/state.json`, 미설정이면 `~/.local/state/omp-harness/state.json`
- Windows: `%LOCALAPPDATA%\omp-harness\state.json`
- 테스트/별도 프로필: `OMP_HARNESS_STATE_FILE` 환경변수로 파일 지정
- 상태 파일 없음/항목 없음: 미설치 표시. 잘못된 JSON/boolean이 아닌 값: 오류를 표시하고 파일 보존.
- 설치됨은 **마지막 기록**이며 실제 환경과 다를 수 있다. 상위 도구가 재검증하지 않는다.

개별 스크립트 실행 시 다음 환경변수를 전달한다.

- `OMP_HARNESS_MANAGER`: 상태 기록용 실행 파일 절대 경로
- `OMP_HARNESS_STATE_FILE`: 사용할 상태 JSON 경로
- `OMP_HARNESS_ITEM_ID`: 선택한 항목 ID
- `OMP_HARNESS_DEFAULTS_FILE`: 배포 기본 설정 파일 경로
- `OMP_HARNESS_SETTINGS_FILE`: 사용자 설정 파일 경로

Linux 스크립트의 성공 지점에서만 기록:

```sh
"$OMP_HARNESS_MANAGER" state set "$OMP_HARNESS_ITEM_ID" true
# 실제 제거가 성공했을 때만 false 기록
"$OMP_HARNESS_MANAGER" state set "$OMP_HARNESS_ITEM_ID" false
```

PowerShell 스크립트:

```powershell
& $env:OMP_HARNESS_MANAGER state set $env:OMP_HARNESS_ITEM_ID true
if ($LASTEXITCODE -ne 0) { throw '상태 기록 실패' }
```

취소/실패 시 상태를 기록하지 않는다. 기존 항목을 유지하면서 선택 ID만 변경한다.
쓰기 잠금과 임시 파일 교체로 동시 기록 충돌/부분 JSON을 피한다. 강제 종료 후 잠금 파일이 남으면
다른 기록 프로세스가 없는 것을 확인하고 해당 `.lock` 파일만 제거한 뒤 재시도한다.

## 사용자 설정 보존

`defaults/settings.json`은 배포 파일이며 현재 기본값은 `{}`다.
사용자 설정은 위 상태 파일 기본 디렉터리의 `settings.json`에 따로 둔다.
관리 도구는 해당 경로만 개별 스크립트에 전달하며, 플러그인별 설정을 자동 적용하지 않는다.
설정을 사용하는 스크립트가 기본값을 읽고 사용자 값을 우선 적용하는 정책을 구현한다.
OMP의 기존 `~/.omp` 설정·로그인·세션은 배포 묶음 밖에 있다. OMP 설정 초기화는 기존 마법사의 명시적 선택에서만 수행한다.

## 배포 파일 부분/전체 교체

Release ZIP을 **설치 경로와 다른 디렉터리**에 풀고 새 묶음의 관리 도구로 적용한다.
ZIP 해제 도구가 실행 권한을 보존하지 않으면 Linux에서는 먼저 `chmod +x manager/manage-linux-*`를 실행한다.

```sh
# 새 ZIP을 푼 디렉터리에서
sh manage.sh bundle apply /absolute/existing/omp-harness      # 전체
sh manage.sh bundle apply /absolute/existing/omp-harness omp  # OMP 설치 스크립트만
# 플러그인 구성요소를 등록한 뒤에는 해당 ID 지정 가능
```

Windows에서도 새 ZIP 디렉터리의 `manage.cmd bundle apply "C:\absolute\existing\omp-harness" omp`를 사용한다.
실행 중인 설치 경로의 관리 도구를 스스로 교체하지 않도록, 설치된 관리 창을 닫고 새 ZIP에서 실행한다.

`bundle.json`에는 구성요소별 버전과 소유 파일/SHA-256을 기록한다.
선택한 구성요소의 해시를 모두 확인한 뒤 적용하고, 이전 manifest에만 있는 해당 구성요소 파일을 제거한다.
부분 교체는 manifest가 있는 기존 설치에서만 가능하며 다른 구성요소와 해당 버전 기록을 유지한다. 전체 교체는 새 묶음에 없는 이전 구성요소의 소유 파일도 제거한다.
이 삭제는 **배포 파일 삭제**이며 그 스크립트가 설치한 프로그램/플러그인의 제거가 아니다. 상태 JSON도 그대로 둔다.
사용자 추가 파일, 상태, 설정은 관리 대상에 넣지 않는다. 새 배포 파일과 기존 사용자 파일의 경로가 충돌하면 적용을 거부한다. 대상 프로그램의 설치 상태는 검사하지 않는다.
일반 파일 쓰기 오류는 이전 파일로 복구를 시도한다. 전원 단절/프로세스 강제 종료까지 복구하는 저널은 아직 없다.

Windows `.exe`를 다시 실행해도 포함된 파일은 갱신되지만 Inno Setup은 이전 manifest의 폐기 파일을 정리하지 않는다.
소유 파일 삭제까지 포함한 정확한 전체/부분 교체에는 위 ZIP 적용 명령을 사용한다.
Linux `.run` 재실행은 같은 manifest 기반 전체 적용을 사용한다.

## 개발: 새 항목 추가

1. `scripts/` 또는 `plugins/<id>/`에 독립 설치/업데이트 스크립트를 작성한다.
2. `catalog.json`에 플랫폼별 entry를 등록한다. 실제 점검/사용자 질문/상태 기록은 스크립트 안에 둔다.
3. `packaging/components.json`에 해당 파일/디렉터리 소유 구성요소를 등록한다.
4. `packaging/VERSION`을 올리고 빌드한다. 같은 파일에 중복 소유자를 지정하면 빌드가 실패한다.
5. 기존 설치에서 부분/전체 갱신 후 사용자 상태·설정 보존을 검사한다.

Go 관리 실행 파일은 빌드할 때 생성해 배포하므로 사용자 PC에 Go/Python을 설치할 필요가 없다.
소스 저장소를 직접 실행하려면 먼저 패키지를 빌드하고 `dist/bundle` 안의 실행 도구를 사용한다.
