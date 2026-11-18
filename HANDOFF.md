# omp-harness 작업 인계

## 현재 목표와 구현

작업 브랜치: `feature/2/normal_develop_설치방식변경및_다른PC에서_복원`.
기준 커밋: `ce66a24d46b284978c4be4117960212d0ed47ee6`, 패키지 버전: **0.3.0**.
DSH의 [최종 작업 요약](https://github.com/kwangyeon1/dsh-harness/blob/fb6e0becf1e867e75ca221372dcca0bef5608724/docs/DAILY_SUMMARY_2026-11-17.md)을 OMP에 맞게 적용했다.

- 일반 설치와 반복 설치 메뉴는 같은 `install-omp.sh`/`.bat`를 호출한다. 별도 TUI 설치 엔진은 호환 진입점으로 교체했다.
- `help ID`, `run ID [ARG...]`, `--project PATH`, 사용법→JSON 인자→확인→실행→메뉴 복귀를 지원한다.
- 관리 목록은 JSON 상태만 표시한다. 성공 상태는 일반 설치 스크립트가 명시적으로 기록한다.
- `snapshot omp BINARY OMP_HOME ENVIRONMENT`는 실제 `omp --version`과 설정 루트의 사용자 파일·잠금 파일을 저장한다.
- `snapshot module ID DIRECTORY ENVIRONMENT`와 모듈별 `restore.sh`/`.ps1`을 지원한다.
- 별도 `restore-tui.sh`/`.bat`, 직접 호출 `restore.sh`/`.bat`를 배포한다.
- OMP 복원은 보관 버전의 공식 바이너리 재설치, JSON/YAML 경로 연결, 내장 Bun의 frozen-lockfile 설치, 실행기 생성 순서다. DSH 소스 빌드/Node/Corepack은 가져오지 않았다.
- `restored.json`은 모든 단계가 성공한 경우에만 생성하며, 기존 경로·같은 스냅샷 ID를 덮어쓰지 않는다. 해시와 symlink를 통한 포함 경로도 검사한다.
- Git push/clone은 사용자 작업이다. OMP 공식 확장 기능을 재구현하지 않는다.

## 주요 파일

| 파일 | 역할 |
| --- | --- |
| `manager/main.go`, `project.go` | 설치 메뉴·인자 전달·로컬 프로젝트·명시적 상태 |
| `manager/restore.go` | 스냅샷·해시·경로 보호·모듈 복원·복원 메뉴 |
| `manager/omp.go` | OMP 릴리스 확인·설정 수집·의존성 확인·바이너리 설치·실행기 |
| `catalog.json` | 일반 설치 entry·파일·사전 조건·인자·예시 |
| `packaging/` | Linux/Windows 패키지 및 소유권·복원 진입점 검사 |
| `docs/INSTALLATION.md`, `RESTORE.md`, `RELEASE.md` | 사용·제한·배포 계약 |

## 검증과 한계

Linux에서 Go 테스트/vet, 패키지 설치·갱신·제거·상태 기록·진입점 검사, Linux/Windows x64·ARM64 관리자 교차 빌드를 수행한다.
자동 테스트는 실제 Git 저장소에 보관→clone, 메뉴 취소/실패 후 반복, 해시 변조·경로·기존 파일 보호,
보관 버전 인자, 잠금 파일 설치, 사용자 확장/설정 유지, 실패 후 성공 기록 방지를 확인한다.
비용이 큰 네트워크 설치는 테스트 실행 파일로 대체하며, 동일 fixture 복원 테스트는 Windows CI에서도 실행한다.

별도 Linux 실검증: 공식 **OMP 18.22.2** 설치→스냅샷→공식 동일 버전 재설치→내장 Bun 잠금 파일 복원→실행기 `--version` 및 `plugin list` 성공.
실제 의존성 표본은 `is-number@7.0.0`이다. 모델 API 호출과 사용자 플러그인의 실제 세션 동작은 검증 범위가 아니다.
Windows 실제 실행은 로컬 Linux 검증에 포함되지 않으며 Windows CI 결과를 별도로 확인한다.

인증 DB·세션·키·캐시·node_modules는 수집하지 않는다. 설정 파일 내 임의 비밀값은 Git 반영 전 검토한다.
Bun 의존성에는 manifest 옆의 `bun.lock`/`bun.lockb`가 필요하다.
외부 `plugin link`·marketplace registry/cache는 별도 모듈 어댑터가 필요하며 미지원 상태를 성공으로 처리하지 않는다.
XDG/별도 agent 경로는 표준 설정 루트로 먼저 모으거나 어댑터를 사용한다. 자세한 제한은 RESTORE.md에 있다.

## 다음 작업

```sh
go test ./manager
go vet ./manager
python3 packaging/build.py linux
python3 packaging/test_linux.py
```

Go 1.24.9/Python 3.10+가 필요하다. `OMP_GO`로 빌드에 사용할 Go 경로를 지정할 수 있다.
Windows 패키지는 Inno Setup 6이 필요하다. `.github/workflows/package.yml`은 `feature/**`에서도 두 OS 검사와 Release 파일 artifact를 만든다.
태그·Release 공개·브랜치 병합은 별도 요청 시 수행한다. 기존 스크린샷은 과거 마법사의 기록이며 현재 UI 검증 자료가 아니다.
