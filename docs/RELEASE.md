# GitHub Release에 업로드할 파일

패키지 버전은 `packaging/VERSION`이며 이번 버전은 **0.3.0**이다. OMP 제품 버전과 별개다.

## 정확한 업로드 목록

| 파일 | 용도 |
| --- | --- |
| `omp-harness-scripts-0.3.0-windows-setup.exe` | Windows 최초 설치/파일 배치 |
| `omp-harness-scripts-0.3.0-linux.run` | Linux 최초 설치/전체 묶음 교체 |
| `omp-harness-bundle-0.3.0.zip` | Windows/Linux x64·ARM64 공용 묶음, 선택 구성요소 또는 전체 적용 |
| `SHA256SUMS` | 위 세 파일의 실제 SHA-256 |

GitHub Actions **Build script installers** 실행의 **omp-harness-release-files** artifact를 다운로드하고
바깥 artifact ZIP을 푼 뒤 위 4개 파일을 같은 Release에 첨부한다.
예시 태그는 `scripts-v0.3.0`이며, 대상은 빌드가 성공한 정확한 커밋으로 지정한다.
Release 본문에는 변경 내용, 위 해시, 설치 명령과 묶음 교체 명령을 적는다.
현재 workflow는 파일 제작/검사/취합만 수행하며 태그나 Release를 자동 생성·공개하지 않는다.

## 묶음 내용

`bundle.json`, `catalog.json`, `manager/`의 플랫폼별 실행 파일, `manage.sh`/`manage.cmd`,
OMP 설치 스크립트, `restore.sh`/`.bat`·`restore-tui.sh`/`.bat`, 복원 안내, 등록된 `scripts/`·`plugins/`, `defaults/`, 제거 스크립트, README, VERSION을 담는다.
`bundle.json`은 빌더가 구성요소별 버전과 파일 해시로 생성한다.
사용자 `state.json`, 사용자 `settings.json`, OMP 인증·세션, 사용자 추가 확장은 포함하지 않는다.
OMP 바이너리는 포함하지 않으며 사용자가 OMP 설치 스크립트를 직접 실행하거나 설치 메뉴에서 확인할 때 공식 서버에서 받는다.

## 직접 빌드

빌드 PC에는 Python 3.10 이상과 Go 1.24.9, Windows에는 Inno Setup 6이 필요하다.

```sh
go test ./manager
python3 packaging/build.py linux
python3 packaging/test_linux.py
```

```powershell
go test ./manager
python packaging/build.py windows
powershell -NoProfile -ExecutionPolicy Bypass -File packaging/test_windows.ps1
```

Go 실행 파일 위치를 지정하려면 `OMP_GO` 환경변수를 사용한다.
두 빌드 결과의 실제 `.exe`, `.run`, `.zip`을 모아 해시를 계산한다. CI에서는 이를 자동 처리한다.
Windows/Linux 각 개별 artifact에는 플랫폼 설치 파일과 universal ZIP, 개별 `.sha256`이 들어 있다.
최종 **omp-harness-release-files** artifact를 사용하면 해시 혼합 없이 Release 업로드 목록을 받을 수 있다.

설치 파일은 아직 코드 서명되지 않았다. SHA-256은 무결성 검사용이며 배포자 서명은 아니다.
현재 비공개 저장소이므로 다운로드에 저장소 접근 권한이 필요하다.

CI는 `normal`, `normal_develop`, `feature/**` push/PR에서 관리자 테스트·vet·패키지 검사를 수행한다. 실제 공식 다운로드 검사는 별도 Installer smoke tests workflow를 수동 실행한다.
