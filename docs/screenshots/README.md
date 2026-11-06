# Codex Cloud에서 실제 설치 및 실행한 화면

- 실행일: 2026-11-06 (Asia/Seoul)
- 실행 환경: Codex Cloud Linux
- 설치 패키지: omp-harness-scripts-0.1.0-linux.run (설치 전 SHA-256 검사 성공)
- 패키지 설치 경로: /workspace/work/omp-demo/scripts
- 실제 OMP 설치 경로: /workspace/work/omp-demo/bin/omp
- 공식 설치 스크립트 실행 후 확인된 버전: omp/18.17.0

## 파일

- omp-screen.png: 실제 OMP 프로세스의 최초 실행 화면. Setup step 1 of 6, 로그인 제공자 선택 단계.
- wizard-screen.png: 패키지에 포함된 대화형 설치 스크립트의 실제 시작 화면.
- GLOBAL-INSTRUCTIONS.md: 결과물을 다운로드 가능하게 전달하도록 요구하는 전역 지침 초안. 계정 설정에 자동 적용되지 않음.

두 PNG는 실제 PTY 프로세스 출력을 로컬 xterm.js 터미널에 연결하고 Chromium/Playwright로 캡처한 원본이다.
데스크톱 전체를 찍은 화면이나 Windows 실행 화면이 아니다. 캡처용 브라우저와 프로세스는 캡처 후 종료했다.
로그인 제공자를 선택하거나 계정 인증을 진행하지 않았고, 모델 대화도 실행하지 않았다.
출력 이미지의 터미널 외부 상단 제목은 캡처 페이지가 표시한 실행 환경 설명이다.

ZIP에는 원본 PNG 2개, 이 설명 파일, 전역 지침 초안이 포함된다.
파일은 비공개 저장소 normal 브랜치에 보관되므로 GitHub 다운로드에는 해당 계정 로그인 및 저장소 권한이 필요하다.
