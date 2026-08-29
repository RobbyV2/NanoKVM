const ja = {
  translation: {
    head: {
      desktop: 'リモートデスクトップ',
      login: 'ログイン',
      changePassword: 'パスワード変更',
      terminal: 'ターミナル',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'ログイン',
      placeholderUsername: 'ユーザー名を入力してください',
      placeholderPassword: 'パスワードを入力してください',
      placeholderCurrentPassword: '現在のパスワード',
      placeholderPassword2: 'パスワードをもう一度入力してください',
      noEmptyUsername: 'ユーザー名は空にできません',
      noEmptyPassword: 'パスワードは空にできません',
      passwordLength: 'パスワードは 8 文字以上 72 文字以下である必要があります',
      noAccount:
        'ユーザー情報の取得に失敗しました。ページを更新してもう一度お試しいただくか、パスワードをリセットしてください。',
      invalidUser: 'ユーザー名またはパスワードが正しくありません',
      locked: 'ログインが多すぎます。後でもう一度お試しください。',
      globalLocked: 'システムは保護されています。後でもう一度試してください。',
      error: '不明なエラー',
      invalidCurrentPassword: '現在のパスワードが正しくありません',
      changePassword: 'パスワード変更',
      changePasswordDesc: 'デバイスのセキュリティのために、パスワードを変更してください！',
      differentPassword: 'パスワードが一致しません',
      illegalUsername: 'ユーザー名に不正な文字が含まれています',
      illegalPassword: 'パスワードに不正な文字が含まれています',
      forgetPassword: 'パスワードを忘れた',
      ok: 'OK',
      cancel: 'キャンセル',
      loginButtonText: 'ログイン',
      tips: {
        reset1: 'パスワードをリセットするには、NanoKVM の BOOT ボタンを 10 秒間押し続けます。',
        reset2: '詳細な手順については、次のドキュメントを参照してください：',
        reset3: 'ウェブデフォルトアカウント：',
        reset4: 'SSH デフォルトアカウント：',
        change1: 'この操作により、以下のパスワードも更新されることに注意してください：',
        change2: 'ウェブログインパスワード',
        change3: 'システム root パスワード（SSH ログインパスワード）',
        change4:
          'パスワードを忘れた場合は、NanoKVM の BOOT ボタンを長押ししてパスワードをリセットする必要があります。'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'NanoKVM の Wi-Fi を設定する',
      success: 'NanoKVM のネットワークステータスを確認するにはデバイスにアクセスしてください。',
      failed: '操作に失敗しました。もう一度お試しください。',
      invalidMode:
        '現在のモードではネットワーク設定はサポートされていません。デバイスで Wi-Fi 設定モードを有効にしてください。',
      confirmBtn: 'OK',
      finishBtn: '完了',
      ap: {
        authTitle: '認証が必要です',
        authDescription: '続行するには、AP パスワードを入力してください',
        authFailed: '無効な AP パスワード',
        passPlaceholder: 'AP パスワード',
        verifyBtn: '確認する'
      }
    },
    screen: {
      scale: '倍率',
      title: '画面',
      video: 'ビデオモード',
      videoDirectTips: 'このモードを使用するには「設定 - デバイス」で HTTPS を有効にしてください',
      resolution: '解像度',
      controlRegion: {
        title: 'マウス位置補正',
        description:
          '操作対象のデバイスが 16:9 以外の解像度を使用していて、カーソルの位置が水平方向または垂直方向にずれる場合に使用します。',
        off: 'オフ',
        auto: '自動',
        autoWarning:
          'ユーザーアプリケーションの背景が完全な黒の場合、補正に失敗することがあります。',
        manual: '手動',
        selectedResolution: '選択領域の解像度',
        unused: '未使用',
        originalResolution: '元の解像度',
        selectResolution: '元の解像度を選択',
        addResolution: 'カスタム解像度を追加',
        add: '追加',
        duplicateResolution: 'この解像度はすでに存在します。',
        width: '幅',
        height: '高さ',
        apply: '計算して適用',
        invalidResolution: 'ビデオの準備完了後、有効な元の解像度を入力してください。',
        select: '領域を選択',
        clear: '自動に戻す',
        saveFailed: '入力領域を保存できませんでした。',
        tooSmall: '選択した領域が小さすぎます。',
        previewUnavailable: 'プレビューを利用できません',
        clearConfirm: '黒帯の自動検出に戻しますか？',
        dragHint: 'ドラッグしてリモートデスクトップの領域を選択',
        finish: '完了',
        confirm: '確認',
        cancel: 'キャンセル'
      },
      auto: '自動',
      autoTips:
        '特定の解像度で画面のちらつきやマウスカーソルのずれが発生する場合があります。リモートホストの解像度を調整するか、自動モードを無効にしてください。',
      fps: 'フレームレート',
      customizeFps: 'カスタマイズ',
      quality: '画質',
      qualityLossless: 'ロスレス',
      qualityHigh: '高',
      qualityMedium: '中',
      qualityLow: '低',
      frameDetect: 'フレーム差分検出',
      frameDetectTip:
        'フレーム間の差異を計算し、リモートホストの画面が変更されない場合はビデオストリームの送信を停止します',
      resetHdmi: 'HDMI をリセット',
      mixedH264: {
        title: 'H.264 ストリームの競合',
        description:
          'H.264 Direct と H.264 WebRTC が同時に使用されています。画面のティアリングや映像の破損が発生する可能性があります。H.264 モードは 1 つだけ使用してください。'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC 接続に失敗しました',
        description: 'ネットワーク接続を確認するか、ビデオモードを切り替えてください。'
      },
      captureStatus: {
        hdmiError: 'HDMI 画面エラー',
        unsupportedResolution: '現在の解像度はサポートされていません',
        retrieving: '画面を取得中...',
        changingResolution: '解像度を切り替え中...',
        updateFailed: '現在、画面を更新できません',
        videoError: '映像表示エラー',
        noHdmi: 'HDMI 信号が検出されません',
        unavailable: '現在、画面を表示できません'
      }
    },
    keyboard: {
      title: 'キーボード',
      paste: '貼り付け',
      tips: '標準的なキーボードの文字と記号のみをサポートしています',
      placeholder: '入力してください',
      submit: '送信',
      virtual: '仮想キーボード',
      readClipboard: 'クリップボードから読み取る',
      clipboardPermissionDenied:
        'クリップボードのアクセス許可が拒否されました。ブラウザでクリップボードへのアクセスを許可してください。',
      clipboardReadError: 'クリップボードの読み取りに失敗しました',
      dropdownEnglish: '英語',
      dropdownGerman: 'ドイツ語',
      dropdownFrench: 'フランス語',
      dropdownRussian: 'ロシア語',
      shortcut: {
        title: 'ショートカット',
        custom: 'カスタム',
        capture: 'ショートカットをキャプチャするにはここをクリックしてください',
        clear: 'クリア',
        save: '保存',
        captureTips:
          'Windows キーなどのシステムレベルのキーを取得するには、全画面表示の許可が必要です。',
        enterFullScreen: '全画面モードに切り替えます。'
      },
      leaderKey: {
        title: 'リーダーキー',
        desc: 'ブラウザの制限を回避して、システムによってブロックされているショートカットキーをリモートホストに送信します。',
        howToUse: '使用方法',
        simultaneous: {
          title: '同時モード',
          desc1: 'リーダーキーを押したまま、ショートカットを押します。',
          desc2:
            '操作は直感的ですが、システムの使用状況により一部のショートカットキーが機能しない場合があります。'
        },
        sequential: {
          title: 'シーケンシャルモード',
          desc1: 'リーダーキーを押す → ショートカットを順番に押す → リーダーキーをもう一度押す。',
          desc2: 'いくつかの手順が必要ですが、システムキーの競合を完全に回避します。'
        },
        enable: 'リーダーキーを有効化',
        tip: 'リーダーキーに設定すると、このキーはショートカットのトリガー専用になり、通常の動作は失われます。',
        placeholder: 'リーダーキーを押してください',
        shiftRight: '右 Shift',
        ctrlRight: '右 Ctrl',
        metaRight: '右 Win',
        submit: '送信',
        recorder: {
          rec: 'REC',
          activate: 'キーを有効化',
          input: 'ショートカットキーを押してください...'
        }
      }
    },
    mouse: {
      title: 'マウス',
      cursor: 'ポインター形状',
      default: 'デフォルトポインター',
      pointer: 'ポインターカーソル',
      cell: 'セルポインター',
      text: 'テキストカーソル',
      grab: 'つかむポインター',
      hide: 'ポインターを非表示',
      mode: 'マウスモード',
      absolute: '絶対モード',
      relative: '相対モード',
      direction: 'ホイール方向',
      scrollUp: '上',
      scrollDown: '下',
      speed: 'ホイール速度',
      fast: '速い',
      slow: '遅い',
      requestPointer:
        '相対モードを使用中です。マウスポインターを取得するには、デスクトップをクリックしてください。',
      resetHid: 'HID をリセット',
      hidOnly: {
        title: 'HID-Only モード',
        desc: '使用中にマウスとキーボードが反応しなくなり、HID をリセットしても効果がない場合は、NanoKVM とデバイス間の互換性に問題がある可能性があります。互換性を向上させるために、HID-Only モードを有効にすることをお勧めします。',
        tip1: 'HID-Only モードを有効にすると、仮想 U ディスクと仮想ネットワークがアンマウントされます',
        tip2: 'HID-Only モードでは、イメージのマウントは無効になります',
        tip3: 'モードを切り替えると、NanoKVM は自動的に再起動します。',
        enable: 'HID-Only モードを有効化',
        disable: 'HID-Only モードを無効化'
      }
    },
    image: {
      title: 'イメージ',
      loading: '読み込み中',
      empty: 'イメージファイルがありません',
      mountMode: 'マウントモード',
      mountFailed: 'マウントに失敗しました',
      mountDesc:
        '一部のシステムでは、イメージをマウントする前にリモートホストで仮想ディスクをアンマウントする必要があります。',
      unmountFailed: 'アンマウントに失敗しました',
      unmountDesc:
        '一部のシステムでは、イメージをアンマウントする前にリモートホストから手動で取り出す必要があります。',
      refresh: 'イメージリストを更新',
      attention: '注意',
      deleteConfirm: 'このイメージを削除してもよろしいですか？',
      okBtn: 'はい',
      cancelBtn: 'いいえ',
      tips: {
        title: 'アップロード方法',
        usb1: 'NanoKVM を USB 経由でコンピュータに接続します；',
        usb2: '仮想ディスクがマウントされていることを確認します（設定 - 仮想ディスク）；',
        usb3: 'コンピュータ上で仮想ディスクを開き、イメージファイルを仮想ディスクのルートディレクトリにコピーします。',
        scp1: 'NanoKVM とコンピュータが同じローカルエリアネットワークに接続されていることを確認します；',
        scp2: 'コンピュータでターミナルを開き、SCP コマンドを使用してイメージファイルを NanoKVM の /data ディレクトリにアップロードします。',
        scp3: '例：scp your-image-path root@your-nanokvm-ip:/data',
        tfCard: 'TF カード',
        tf1: 'この方法は Linux システムでサポートされています',
        tf2: 'NanoKVM から TF カードを取り出します（フルバージョンでは、まずケースを分解してください）；',
        tf3: 'TF カードをカードリーダーに挿入してコンピュータに接続します；',
        tf4: 'コンピューターから TF カードの /data ディレクトリにイメージファイルをコピーします；',
        tf5: 'TF カードを NanoKVM に挿入します。'
      }
    },
    script: {
      title: 'スクリプト',
      upload: 'アップロード',
      run: '実行',
      runBackground: 'バックグラウンドで実行',
      runFailed: '実行に失敗しました',
      attention: '注意',
      delDesc: 'このファイルを削除してもよろしいですか？',
      confirm: 'はい',
      cancel: 'いいえ',
      delete: '削除',
      close: '閉じる'
    },
    terminal: {
      title: 'ターミナル',
      nanokvm: 'NanoKVM ターミナル',
      serial: 'シリアルポートターミナル',
      serialPort: 'シリアルポート',
      serialPortPlaceholder: 'シリアルポートを入力してください',
      baudrate: 'ボーレート',
      parity: 'パリティ',
      parityNone: 'なし',
      parityEven: '偶数',
      parityOdd: '奇数',
      flowControl: 'フロー制御',
      flowControlNone: 'なし',
      flowControlSoft: 'ソフトウェア',
      flowControlHard: 'ハードウェア',
      dataBits: 'データビット',
      stopBits: 'ストップビット',
      confirm: 'OK'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'コマンドを送信中...',
      sent: 'コマンドを送信しました',
      input: 'MAC アドレスを入力してください',
      ok: 'OK'
    },
    download: {
      title: 'イメージダウンローダー',
      input: 'リモートイメージの URL を入力してください',
      ok: 'OK',
      disabled:
        '/data パーティションは読み取り専用であり、イメージのダウンロードには使用できません',
      uploadbox: 'ここにファイルをドロップするか、クリックして選択してください',
      inputfile: '画像ファイルを入力してください',
      NoISO: 'ISO なし',
      sha256: 'SHA-256（任意）',
      sha256Placeholder: '64 文字の SHA-256 チェックサムを入力してください',
      invalidSHA256: 'SHA-256 は 64 文字の 16 進数文字列である必要があります',
      failed: 'ダウンロードに失敗しました',
      success: 'ダウンロードに成功しました',
      checksumFailed: 'ダウンロードに失敗しました：SHA-256 検証に失敗しました',
      cancel: 'キャンセル',
      cancelFailed: 'ダウンロードのキャンセルに失敗しました'
    },
    power: {
      title: '電源',
      showConfirm: '確認メッセージ',
      showConfirmTip: '電源操作の確認メッセージを表示する',
      reset: 'リセット',
      power: '電源',
      powerShort: '電源（クリック）',
      powerLong: '電源（長押し）',
      resetConfirm: '再起動を実行しますか？',
      powerConfirm: '電源操作を実行しますか？',
      okBtn: 'はい',
      cancelBtn: 'いいえ'
    },
    devices: {
      title: 'デバイス',
      stale: 'デバイスのライブ状態を取得できません。再接続しています。',
      empty:
        'カメラ、マイクまたはスピーカーのスロットが設定されていません。「設定 - デバイス」で追加してください。',
      available: '利用可能',
      waiting: 'ホストがソースを待っています',
      hostOpen: 'ホスト受信中',
      hostIdle: 'ホスト待機中',
      hostPlaying: 'ホストが音声を再生しています',
      hostSending: 'ホスト再生中',
      sending: 'このブラウザーから送信中',
      receiving: 'このブラウザーで再生中',
      black: '黒画面',
      silence: 'デジタル無音',
      resuming: '再開待ち',
      stop: '共有を停止',
      stopListening: '再生を停止',
      disconnect: '切断',
      takeover: '引き継ぐ',
      refused: '{{source}} の {{owner}} が使用中',
      connectedSources_one: '接続中のソース {{count}} 件',
      connectedSources_other: '接続中のソース {{count}} 件',
      connectedSources: '接続中のソース {{count}} 件',
      connection: {
        connecting: '接続中',
        connected: 'ライブ',
        disconnected: '再接続中'
      },
      share: {
        camera: 'カメラを共有',
        microphone: 'マイクを共有',
        speaker: '音声を再生',
        usbDevice: 'USB を共有'
      },
      permission: {
        denied: 'ブラウザーのサイト設定でブロックされています',
        prompt: 'ブラウザーが許可を求めます',
        insecure:
          'このページは HTTPS で配信されていないため、ブラウザーがこのデバイスをブロックしています。「設定 - ネットワーク」で HTTPS を有効にしてください。'
      },
      capture: {
        unsupported: 'このブラウザーは音声も映像も取り込めません',
        camera: 'このブラウザーはカメラ映像をエンコードできません',
        microphone: 'このブラウザーはマイクの音声を処理できません',
        speaker: 'このブラウザーは音声を再生できません'
      },
      mic: {
        mute: 'ミュート',
        unmute: 'ミュート解除'
      },
      revoked: {
        released: '共有が停止されました',
        lease_expired: 'このブラウザーが戻る前にリースが期限切れになりました',
        admin_disconnect: '管理者がすべてのソースを切断しました',
        slot_removed: 'スロットが削除されました',
        slot_changed: 'スロットが再設定されました',
        taken_over: '管理者がこのスロットを引き継ぎました'
      },
      usb: {
        surrendered: 'USB パススルーがキーボードとマウスを保持しています',
        surrenderedDesc:
          'リモートホストには NanoKVM のキーボード、マウス、仮想メディアではなくインポートされたデバイスが見えます。セッションが終了すると元に戻ります。',
        unsupported: 'WebUSB には Chromium 系ブラウザーが必要です',
        insecure:
          'このページは HTTPS で配信されていないため、ブラウザーが WebUSB を無効にしています。「設定 - ネットワーク」で HTTPS を有効にしてください。',
        session: '{{device}} を転送中（{{mode}}）',
        idle: 'パススルーセッションはありません',
        mode: {
          hybrid: 'ハイブリッド',
          exact: '完全一致'
        }
      }
    },
    settings: {
      title: '設定',
      display: {
        title: 'ディスプレイ',
        loading: '読み込み中...',
        active: '現在の EDID',
        activeUnknown:
          'NanoKVM は起動してから EDID を書き込んでいないため、ホストが認識しているモニター情報は不明です。',
        appliedAt: '適用日時 {{time}}',
        download: 'ダウンロード',
        downloadBackup: '以前のものをダウンロード',
        preset: 'モニタープリセット',
        presetPlaceholder: 'モニターを選択',
        upload: 'アップロード',
        selected: '選択した EDID',
        errors: 'エラー',
        warnings: '警告',
        info: '情報',
        unknownMonitor: '不明なモニター',
        edidVersion: 'EDID {{version}}',
        audioYes: '音声あり',
        audioNo: '音声なし',
        extensionBlocks: '拡張ブロック: {{blocks}}',
        apply: '適用',
        applyTitle: 'この EDID を適用しますか？',
        before: '現在',
        after: '新規',
        hdmiNotice: 'EDID の書き込み中は映像キャプチャが停止し、完了後に自動で再開します。',
        powerCycleNotice:
          '新しい EDID を有効にするには、本体の電源ケーブルを物理的に抜き差しする必要があります。',
        powerCycleUnverified:
          '書き込みを検証できなかったため、この装置の電源を物理的に抜き差しするまで、ビデオチップは今保持している内容をそのまま持ち続けます。',
        applied: 'EDID を適用し、検証しました。',
        applyFailed: 'EDID の適用に失敗しました。',
        busy: '映像チップがビジー状態でした。もう一度お試しください。',
        unsupported: 'このデバイスは EDID の変更に対応していません。',
        toolMissing: 'このファームウェアには EDID ツールがありません。',
        noAudio: 'この EDID は音声を通知しないため、ホストが音声の送出を停止する場合があります。',
        oldVersion: 'この EDID は 1.4 より前のバージョンを使用しています。',
        interlaced: '推奨タイミングはインターレースです。',
        tooLarge:
          '推奨タイミングが 1920x1080 60 Hz を超えており、NanoKVM のキャプチャ能力を上回ります。',
        recovery: '復旧',
        recoveryNeeded:
          '直前の書き込みが検証されなかったため、映像チップの EDID 領域は不明な状態です。工場出荷時の EDID を復元して、既知の状態に戻してください。',
        recoveryDesc:
          '適用した EDID でホストの画面が出なくなった場合に、既知の EDID を映像チップへ書き戻します。',
        restoreFactory: '工場出荷時の EDID を復元',
        restoreBackup: '以前の EDID を復元',
        restoreTitle: 'この EDID を復元しますか？',
        restoreFactoryTarget: 'NanoKVM に同梱されている工場出荷時の EDID です。',
        restoreBackupTarget: '最新のバックアップです（適用日時 {{time}}）。',
        restoreNotice: '復元は適用と同じ方法で書き込まれ、同じ影響があります。',
        restored: 'EDID を復元し、検証しました。',
        restoreFailed: 'EDID の復元に失敗しました。',
        mismatchTitle: '書き込んだ内容と読み戻した内容',
        mismatchWritten: '書き込み',
        mismatchRead: '読み戻し',
        restoreOkBtn: '復元',
        hardware: '検出されたハードウェア: {{hardware}}',
        hardwareUnknown: '不明',
        confirmWord: '適用',
        confirmPrompt: '適用ボタンを有効にするには {{word}} と入力してください。',
        okBtn: '適用',
        cancelBtn: 'キャンセル'
      },
      presentation: {
        title: 'USB プレゼンテーション',
        loading: '読み込み中...',
        current: '現在の USB プレゼンテーション',
        noProfile: '適用されたプロファイルはありません',
        linked: 'リンクされた機能',
        hostState: 'ホスト側 USB',
        hostUnbound: 'コントローラーは未バインド',
        hdmiState: 'HDMI 入力',
        hdmiSignal: '信号あり',
        hdmiUnreported: 'キャプチャの報告はまだありません',
        endpoints: 'エンドポイント',
        fifos: 'FIFO スロット',
        pending: '保留中の変更',
        pendingEdits: '未保存のアイデンティティ編集',
        pendingProfile: '{{profile}} は選択されていますが適用されていません',
        pendingNone: 'なし',
        lastApply: '最後の適用',
        applyFailed: '{{time}} に {{profile}} で失敗',
        applyClean: '記録された失敗はありません',
        lastKnownGood: '最後に正常だった構成',
        rollbackTarget: 'ロールバック先',
        rollbackNone: 'なし',
        powerCyclePending:
          'コントローラーがホストから取り上げられました。デバイスを取り戻すには、接続したコンピューターの電源を入れ直してください。',
        rollback: 'ロールバック',
        rollbackTitle: '{{profile}} にロールバックしますか？',
        rollbackDesc: 'ガジェットが再列挙され、USB 機能が一時的に切断されます。',
        profile: 'USB プロファイル',
        builtIn: '組み込み',
        descriptors: 'ディスクリプター',
        imported: 'インポート済み',
        clone: '複製',
        cloneTitle: 'このプロファイルを複製',
        cloneToEdit:
          '組み込みプロファイルは読み取り専用のままです。アイデンティティを編集するにはこのプロファイルを複製してください。',
        profileName: 'プロファイル名',
        profileNameHint: '英小文字、数字、ピリオド、アンダースコア、ハイフンが使えます。',
        import: 'パッケージをインポート',
        export: 'パッケージをエクスポート',
        delete: '削除',
        deleteTitle: 'このプロファイルを削除しますか？',
        deleteDesc: '{{profile}} を NanoKVM から削除します。',
        identity: 'USB アイデンティティ',
        preset: 'プリセットのアイデンティティ',
        presetPlaceholder: '既知のデバイスからアイデンティティをコピー',
        presetHint:
          'プリセットは Vendor ID、Product ID、そして 2 つの名前欄を埋めます。ディスクリプターは付いてきません。',
        presetSource: '{{source}} に記録されているとおりのアイデンティティ',
        vendorId: 'Vendor ID',
        foreignVendor: 'この Vendor ID は別のメーカーのものです',
        productId: 'Product ID',
        bcdUSB: 'USB バージョン',
        bcdDevice: 'デバイスバージョン',
        manufacturer: 'メーカー',
        product: '製品名',
        serial: 'シリアル番号',
        configuration: 'コンフィギュレーション文字列',
        hidLayout: 'HID デバイス',
        hidRoleKeyboard: 'キーボード',
        hidRoleRelative: 'マウス（相対）',
        hidRoleAbsolute: 'ポインター（絶対）',
        hidOff: 'なし',
        hidInterface: 'インターフェース {{index}}',
        hidBootKeyboardShared:
          'キーボードがインターフェースを共有するため、boot プロトコルのレポートを提供しなくなります。一部の BIOS や UEFI では認識されません。',
        functions: '機能',
        descriptorAssets: '保存済みのディスクリプターファイル: {{count}}',
        endpointUse: 'IN 使用 {{inUse}}、空き {{inFree}}／OUT 使用 {{outUse}}、空き {{outFree}}',
        apply: '適用',
        applyTitle: 'この USB プロファイルを適用しますか？',
        applyDesc: 'NanoKVM は接続されたコンピューターに {{profile}} を提示します。',
        reconnect:
          'ガジェットが再バインドされる間、キーボード、マウス、その他の USB 機能が一時的に切断されます。',
        applyLinks: 'リンクする機能: {{functions}}',
        applyRemoves: '削除する機能: {{functions}}',
        applyNoHid: 'この適用の後に HID 機能は残りません。キーボードとマウスは動作しなくなります。',
        applyRollback: '適用に失敗した場合は {{profile}} に戻ります。',
        recoveryPowerCycle:
          'この適用では HID がひとつも残らないため、応答しなくなったホストは電源の入れ直しでしか復旧できません。',
        recoveryReboot:
          '複合デバイスからインターフェイスがひとつ消えるため、残りを再バインドするにはホストの再起動が必要になる場合があります。',
        recoveryHdmiReset:
          'ビデオ機能が作り直されるため、その背後にあるキャプチャ経路もリセットされます。',
        recoveryReconnect: 'ホストがデバイスを再列挙し、USB 機能が一時的に切断されます。',
        cancel: 'キャンセル',
        noFunctions: 'リンクされた機能はありません',
        loadFailed: 'プレゼンテーションプロファイルを読み込めませんでした',
        operationFailed: 'プレゼンテーションの操作に失敗しました'
      },
      passthrough: {
        title: 'USB パススルー',
        loading: '読み込み中...',
        mode: 'モード',
        hybrid: 'ハイブリッド',
        exact: '完全',
        hybridDesc: '対応デバイス向けに、boot キーボードと相対マウスを残します。',
        exactDesc: 'NanoKVM のすべての USB 機能をパススルーしたデバイスに置き換えます。',
        hybridWarning: 'ハイブリッドはキーボードと相対マウスを使えるまま残します',
        hybridWarningDesc:
          'パススルーした機能が有効な間、ストレージ、USB ネットワーク、絶対ポインターは切断されます。',
        hidWarning: 'パススルーを開始するとキーボード、マウス、仮想メディアを手放します',
        hidWarningDesc:
          'NanoKVM の USB デバイスコントローラーは 1 つだけで、プロキシがそれを占有します。そのためセッション中、リモートホストには NanoKVM のキーボード・マウス・仮想メディアではなく、パススルーされたデバイスが見えます。セッションを停止すればそれらは自動的に戻ります。この Web 画面は影響を受けないため、いつでもこのページからセッションを停止できます。',
        hidWarningSafeDesc:
          'NanoKVM の USB デバイスコントローラーは 1 つだけで、プロキシがそれを占有します。そのためセッション中、リモートホストには NanoKVM のキーボード・マウス・仮想メディアではなく、パススルーされたデバイスが見えます。セッションを停止すれば戻ります。',
        isoLabel: 'アイソクロナス転送を許可',
        isoHint:
          'ウェブカメラ、マイクなどのストリーミングデバイスを通します。このハードウェアで何が出せるかは誰も測定していません。',
        isoWarning:
          'アイソクロナス転送はここでは未検証で、セッションを停止するまでキーボードとマウスを奪う可能性があります',
        info: {
          title: '情報',
          hybrid:
            'ハイブリッドモードはキーボードと相対マウスを使えるまま残します。パススルーしたデバイスが有効な間、ストレージ、USB ネットワーク、絶対ポインターは切断されます。',
          exact:
            '完全モードは NanoKVM のすべての USB 機能をパススルーしたデバイスに置き換えます。キーボード、マウス、仮想メディアはセッションを停止すると自動的に戻ります。',
          udc: 'NanoKVM の USB デバイスコントローラーは 1 つだけで、プロキシがそれを占有します。そのためセッション中は上記の機能が使えなくなります。',
          web: 'この Web 画面は影響を受けないため、いつでもこのページからセッションを停止できます。',
          network:
            'パススルーはイーサネットか Wi-Fi 経由で開始してください。NanoKVM の USB ネットワークからの開始は、その接続自体が失われるため拒否されます。',
          iso: 'ウェブカメラ、マイクなどのアイソクロナスデバイスは、アイソクロナス転送を許可するまで拒否されます。この経路は動作しますが、このハードウェアで測定されたことがないため、スループットは未知として扱ってください。',
          camera:
            'デバイスにあるブラウザーのカメラとマイクは、リモートホストにそれらを渡す実績のある方法です。'
        },
        session: 'セッション',
        activeDesc: 'デバイスがインポートされ、プロキシが USB コントローラーを保持しています。',
        inactiveDesc:
          'セッションはありません。キーボード、マウス、仮想メディアは通常どおり動作しています。',
        device: 'デバイス',
        busId: 'バス ID',
        speed: '速度',
        exporter: 'エクスポーター',
        local: 'インポート先',
        localValue: 'バス {{bus}}、アドレス {{address}}',
        udc: 'USB コントローラー',
        pid: 'プロキシ PID',
        startedAt: '開始時刻',
        isoDevice:
          'このデバイスはアイソクロナスエンドポイントでストリーミングします。このハードウェアでは未測定です',
        exporterLabel: 'エクスポーターのアドレス',
        exporterHint:
          'NanoKVM が接続するホストとポートです。下のトンネルを使う場合は {{exporter}} になります。',
        busIdLabel: 'お使いのマシン上のバス ID',
        busIdHint: 'usbip list -l がそのデバイスに対して表示する busid です（例: {{example}}）。',
        start: 'パススルーを開始',
        stop: 'パススルーを停止',
        startTitle: 'USB パススルーを開始しますか？',
        startDevice: 'NanoKVM は {{exporter}} から {{busId}} をインポートします。',
        startHid:
          'セッション中は USB キーボード、マウス、仮想メディアが使えなくなり、停止すると自動的に戻ります。',
        startIso:
          'ウェブカメラなどのアイソクロナス転送デバイスは、開始前にアイソクロナス転送のスイッチを入れる必要があります。',
        startWeb:
          'この Web 画面は動作し続けるため、いつでもこのページからセッションを停止できます。',
        startNetwork:
          'このページはイーサネットか Wi-Fi 経由で使用してください。NanoKVM の USB ネットワークからの開始は、その接続自体が失われるため拒否されます。',
        okBtn: '開始',
        cancelBtn: 'キャンセル',
        instructions: 'お使いのマシンでの操作',
        instructionsDesc:
          '設計上、クライアントエージェントのインストールは不要です。デバイスが接続されているマシンで、標準の usbip コマンドを次のとおり実行してください。',
        copyFailed: 'コピーできませんでした。コマンドを手動でコピーしてください。',
        copyInsecure:
          'このページは HTTPS で配信されていないため、ブラウザーがコピーを拒否しました。コマンドを手動でコピーするか、「設定 - ネットワーク」で HTTPS を有効にしてください。',
        directNote:
          'トンネルを使わない場合、usbipd をネットワーク上から到達可能にし、上のエクスポーターのアドレスにそれを指定する必要があります。usbip はデバイスの通信を暗号化しないため、トンネルの利用を推奨します。',
        steps: {
          modprobe: {
            title: 'エクスポーター側のドライバーを読み込む',
            desc: 'usbip-host はローカルデバイスを引き渡すためのモジュールで、既定では読み込まれていません。'
          },
          list: {
            title: 'デバイスを探す',
            desc: 'ローカルのデバイスを busid とベンダー:プロダクト ID 付きで一覧表示します。目的のデバイスの busid を控えてください。'
          },
          bind: {
            title: 'usbip にバインドする',
            desc: 'デバイスを本来のドライバーから切り離すため、アンバインドするまでこのマシンでは使えなくなります。'
          },
          serve: {
            title: '公開する',
            desc: 'usbipd はフォアグラウンドで動作し続け、NanoKVM がデバイスをインポートするのを待ちます。',
            notice:
              '標準の usbipd には待ち受けアドレスの指定がなく、すべてのインターフェースで待ち受けます。ファイアウォールでポート {{port}} を閉じ、下のトンネルからのみ届くようにしてください。'
          },
          tunnel: {
            title: 'NanoKVM に向ける',
            desc: 'SSH のリバーストンネルです。NanoKVM 自身のループバックのポート {{port}} が、このマシンの usbipd になります。セッション中は起動したままにしてください。'
          },
          exporter: {
            title: 'これをエクスポーターに指定する',
            desc: '上のエクスポーター欄にこれを入力し、バス ID を入れてセッションを開始します。'
          },
          unbind: {
            title: 'デバイスを戻す',
            desc: 'セッションを停止したあと、このコマンドでデバイスを本来のドライバーに返します。'
          }
        }
      },
      mcp: {
        title: 'MCP サービス',
        service: 'MCP リモート制御',
        serviceDesc:
          '信頼できる MCP クライアントによるキーボードとマウスの操作、およびスクリーンショットの取得を許可します',
        securityWarning:
          'この API キーを持つ人は誰でもリモートホストを操作し、画面を表示できます。HTTPS を使用し、信頼できるネットワークでのみ有効にしてください。',
        endpoint: 'エンドポイント',
        apiKey: 'API キー',
        regenerateConfirmTitle: 'MCP API キーを再生成しますか？',
        regenerateConfirmDesc: '現在のキーは直ちに使用できなくなります。',
        enableConfirmTitle: '外部 MCP 制御を有効にしますか？',
        enableConfirmDesc:
          'MCP を有効にすると PicoClaw が停止し、アクティブな PicoClaw セッションがすべて終了します。',
        failed: 'MCP 操作に失敗しました',
        copyFailed: 'コピーに失敗しました。手動でコピーしてください。',
        copyInsecure:
          'このページは HTTPS で配信されていないため、ブラウザーがコピーを拒否しました。手動でコピーするか、「設定 - ネットワーク」で HTTPS を有効にしてください。',
        okBtn: '確認',
        cancelBtn: 'キャンセル'
      },
      about: {
        title: 'NanoKVM について',
        information: '情報',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'アプリケーションバージョン',
        applicationTip: 'NanoKVM ウェブアプリケーションバージョン',
        image: 'イメージバージョン',
        imageTip: 'NanoKVM システムイメージバージョン',
        deviceKey: 'デバイスキー',
        community: 'コミュニティ',
        hostname: 'ホスト名',
        hostnameUpdated: 'ホスト名は正常に変更され、再起動後に有効になります',
        ipType: {
          Wired: '有線',
          Wireless: 'ワイヤレス',
          Other: 'その他'
        }
      },
      appearance: {
        title: '外観',
        display: '表示',
        language: '言語',
        languageDesc: 'インターフェース言語の選択',
        webTitle: 'ウェブページタイトル',
        webTitleDesc: 'ウェブページタイトルのカスタマイズ',
        favicon: 'ファビコン',
        faviconDesc: 'ブラウザタブのアイコンをカスタマイズ',
        faviconPlaceholder: '画像の URL',
        faviconUpload: 'アップロード',
        faviconReset: 'リセット',
        faviconCustom: 'カスタムアイコン',
        faviconBoot: '/boot/logo.ico のアイコン',
        faviconDefault: '既定のアイコン',
        faviconOverridesBoot: '/boot/logo.ico を上書きしています',
        faviconErrUrl: 'http:// または https:// の画像アドレスを入力してください',
        faviconErrFetch: 'デバイスが画像をダウンロードできませんでした',
        faviconErrLarge: '画像が大きすぎます。上限は 256 KB です',
        faviconErrType: '対応していない画像です。.ico、.png、.svg、.gif、.jpg を使用してください',
        faviconErrSave: 'アイコンを保存できませんでした',
        menuBar: {
          title: 'メニューバー',
          mode: '表示モード',
          modeDesc: 'メニューバーの画面表示方法',
          modeOff: '閉じる',
          modeAuto: '自動非表示',
          modeAlways: '常に表示',
          keyboardLedStatus: 'キーボードロックの表示',
          keyboardLedStatusDesc:
            'リモートコンピューターの Num Lock、Caps Lock、Scroll Lock の状態を表示',
          icons: 'メニューアイコン',
          iconsDesc: 'メニューバーでのサブメニューアイコンの表示'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'リモートキーボードのロック状態',
        indicatorLabel: '{{label}}：{{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'オン',
        off: 'オフ',
        unknown: '不明'
      },
      device: {
        title: 'デバイス',
        oled: {
          title: 'OLED',
          description: 'OLED 画面の自動スリープ時間',
          0: '無効',
          15: '15秒',
          30: '30秒',
          60: '1分',
          180: '3分',
          300: '5分',
          600: '10分',
          1800: '30分',
          3600: '1時間'
        },
        ssh: {
          description: 'SSH リモートアクセスを有効にする',
          tip: '使用する前に必ず強力なパスワードを設定してください（アカウント - パスワードの変更）'
        },
        advanced: '詳細設定',
        swap: {
          title: 'スワップ',
          disable: '無効',
          description: 'スワップファイルのサイズを設定する',
          tip: 'この機能を有効にすると、SD カードの寿命が短くなる可能性があります！'
        },
        mouseJiggler: {
          title: 'マウスジグラー',
          description: 'リモートホストの休止を防ぐ',
          disable: '閉じる',
          absolute: '絶対モード',
          relative: '相対モード'
        },
        mdns: {
          description: 'mDNS 検出サービスを有効にする',
          tip: 'この機能を使用していない場合は、オフにすることをお勧めします'
        },
        hdmi: {
          description: 'HDMI/モニター 出力機能を有効にする',
          idleTimeoutTitle: 'キャプチャのアイドルタイムアウト',
          idleTimeoutDescription:
            'アクティブな閲覧者がいない状態が次の時間続いたら HDMI キャプチャを停止',
          minutes: '分'
        },
        autostart: {
          title: '自動起動スクリプト設定',
          description: 'NanoKVM の起動時に自動的に実行されるスクリプトファイルを管理します',
          new: '新しいスクリプトを作成する',
          deleteConfirm: 'このファイルを削除してもよろしいですか？',
          yes: 'はい',
          no: 'いいえ',
          scriptName: '自動起動スクリプト名',
          scriptContent: '自動起動スクリプト内容',
          settings: '設定'
        },
        hidOnly: 'HID-Only モード',
        hidOnlyDesc:
          'このモードでは仮想デバイスはマウントされなくなり、基本的な HID 制御機能のみが保持されます。',
        disk: '仮想ディスク',
        diskDesc: 'リモートホストに仮想 USB ドライブをマウントする',
        rebindNotice:
          'このスイッチを切り替えると USB デバイスは再列挙され、ターゲットは仮想デバイスと USB ネットワークを一時的に失います。',
        media: {
          title: 'カメラ・マイク・スピーカーのスロット',
          desc: '接続先のホストに見えるメディアデバイスを宣言します。エンドポイントの余裕は USB プロファイルの適用時に確認されます。 保存するとデバイスが再列挙され、接続中のブラウザーは切断されます。',
          cameras: 'カメラ',
          microphones: 'マイク',
          speakers: 'スピーカー',
          name: '名前',
          namePlaceholder: '接続先のホストに表示されます',
          addCamera: 'カメラを追加',
          addMicrophone: 'マイクを追加',
          addSpeaker: 'スピーカーを追加',
          remove: '削除',
          cameraDefault: 'カメラ {{index}}',
          microphoneDefault: 'マイク {{index}}',
          speakerDefault: 'スピーカー {{index}}',
          nameRequired: 'すべてのスロットに名前が必要です。',
          budgetHint:
            '6 つの USB IN エンドポイントは固定のハードウェア上限です。「USB プレゼンテーション」でキーボード・マウス・絶対座標を 1 つの HID インターフェイスにまとめるか、ここで仮想ディスクを、または「ネットワーク」で USB ネットワークアダプターをオフにしてください。',
          unsupported:
            'このカーネルはメディアデバイスに名前を付けられないため、ホストには既定の名前が表示されます。',
          save: 'スロットを保存',
          disconnect: '切断',
          disconnectAll: 'すべてのソースを切断',
          limit: 'カメラ、マイク、スピーカーのスロットは合計 8 個以下にしてください。',
          failed: 'メディアスロットを更新できませんでした。'
        },
        reboot: '再起動',
        rebootDesc: 'NanoKVM を再起動してもよろしいですか?',
        okBtn: 'はい',
        cancelBtn: 'いいえ'
      },
      network: {
        title: 'ネットワーク',
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi 設定',
          apMode: 'AP モードが有効になりました。QR コードをスキャンして Wi-Fi に接続してください。',
          connect: 'Wi-Fi に接続',
          connectDesc1: 'SSID とパスワードを入力してください',
          connectDesc2: 'このネットワークに接続するためのパスワードを入力してください',
          disconnect: 'このネットワーク接続を切断しますか？',
          failed: '接続に失敗しました。もう一度お試しください。',
          ssid: 'SSID',
          password: 'パスワード',
          joinBtn: '接続',
          confirmBtn: 'OK',
          cancelBtn: 'キャンセル'
        },
        tls: {
          description: 'HTTPS プロトコルを有効にする',
          tip: '注意：HTTPS を使用すると、特に MJPEG ビデオモードで遅延が増加する可能性があります。'
        },
        usb: {
          title: 'USB ネットワークアダプター',
          desc: '制御対象のコンピューターに USB 経由でネットワークカードを与えます',
          type: 'アダプターの種類',
          typeDesc: '最新のシステムには NCM、古い Windows には RNDIS'
        },
        bridge: {
          title: 'アダプターの接続先',
          lan: '自分のネットワーク',
          kvmOnly: 'NanoKVM のみ',
          lanDesc:
            'コンピューターは NanoKVM の Ethernet ポート経由で自分のネットワークに参加し、ルーターから固有のアドレスを受け取ります。',
          kvmOnlyDesc:
            'コンピューターは NanoKVM からアドレスを受け取り、NanoKVM には届きますが、その先へは届きません。',
          loading: '読み込み中...',
          state: '状態',
          states: {
            disabled: 'NanoKVM のみ',
            enabled: '自分のネットワーク',
            rolledBack: 'ロールバック済み',
            failed: '失敗',
            pending: '実行中'
          },
          uplink: 'アップリンク',
          ports: 'ポート',
          up: 'リンクアップ',
          down: 'リンクダウン',
          noLink: 'リンクなし',
          enableTitle: 'コンピューターを自分のネットワークに接続しますか？',
          disableTitle: 'コンピューターを NanoKVM のみに限定しますか？',
          reconnect: 'アドレスの移動中、管理接続は一時的に切断されてから再接続します。',
          rollback: '検証に失敗した場合、以前の構成が自動的に復元されます。',
          enableBtn: '自分のネットワークに接続',
          disableBtn: 'NanoKVM のみ',
          cancelBtn: 'キャンセル',
          interrupted: '適用中に接続が中断されました。現在の状態を再確認しています。',
          pendingNotice: 'ブリッジの変更がまだ実行中か、完了前に中断されました。',
          revert: '以前の構成に戻す',
          rolledBackNotice: '前回の変更はロールバックされ、以前の構成が復元されました。',
          verifyFailed: '検証に失敗しました: {{gates}}',
          gates: {
            address: 'アドレス',
            gateway: 'ゲートウェイ',
            inbound: '受信接続'
          },
          inboundWeak:
            '受信確認は NanoKVM が自分自身に接続することでのみ成立しました。これは Web サービスが待ち受けていて本体から到達できることを示すだけで、ネットワークからの要求が届くことを示すものではありません。',
          noCarrier:
            '{{port}} にリンクがありません。ケーブルを接続するまで、ブリッジからネットワークへの経路はありません。',
          loop: 'ルーターが {{port}} でも学習されています。つまりそのポートは同じネットワークへの二つ目の経路です。スパニングツリーは無効なので、ここでループを断ち切るものはありません。どちらか一方の経路を外してください。',
          failedNotice:
            '前回の変更を取り消せませんでした。NanoKVM には Wi-Fi AP またはシリアルコンソール経由でしかアクセスできない可能性があります。'
        },
        dns: {
          title: 'DNS',
          description: 'NanoKVM の DNS サーバーを設定',
          mode: 'モード',
          dhcp: 'DHCP',
          manual: '手動',
          add: 'DNS を追加',
          save: '保存',
          invalid: '有効な IP アドレスを入力してください',
          noDhcp: '現在 DHCP DNS は利用できません',
          saved: 'DNS 設定を保存しました',
          saveFailed: 'DNS 設定の保存に失敗しました',
          unsaved: '未保存の変更',
          maxServers: 'DNS サーバーは最大 {{count}} 個までです',
          dnsServers: 'DNS サーバー',
          dhcpServersDescription: 'DNS サーバーは DHCP から自動取得されます',
          manualServersDescription: 'DNS サーバーは手動で編集できます',
          networkDetails: 'ネットワーク詳細',
          interface: 'インターフェイス',
          ipAddress: 'IP アドレス',
          subnetMask: 'サブネットマスク',
          router: 'ルーター',
          none: 'なし'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC サーバー',
        description:
          'NanoKVM のアカウントでログインすれば、任意の VNC クライアントからリモート画面を表示し、キーボードとマウスを使用できます',
        port: 'ポート',
        portDescription: 'NanoKVM のアドレスのこのポートに接続します'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'メモリ最適化',
          tip: 'メモリ使用量が上限を超えると、メモリ解放のためにより積極的にガベージコレクションが実行されます。Tailscale を使用する場合は 50MB に設定することをお勧めします。この設定を有効にするには Tailscale を再起動する必要があります。'
        },
        swap: {
          title: 'スワップメモリ',
          tip: 'メモリ最適化を有効にしても問題が解決しない場合は、スワップメモリ​​を有効にしてみてください。有効にするとスワップファイルが 256MB に設定されます。このサイズは「設定 - デバイス」で変更できます。'
        },
        restart: 'Tailscale を再起動しますか？',
        stop: 'Tailscale を停止しますか？',
        stopDesc: 'Tailscale からログアウトし、起動時の自動実行を無効にします。',
        loading: '読み込み中...',
        notInstall: 'Tailscale が見つかりません。インストールしてください。',
        install: 'インストール',
        installing: 'インストール中',
        failed: 'インストールに失敗しました',
        retry: 'ページを更新してもう一度お試しいただくか、手動でインストールしてください',
        download: 'ダウンロードして',
        package: 'インストールパッケージを',
        unzip: '解凍してください',
        upTailscale: 'tailscale ファイルを NanoKVM の /usr/bin ディレクトリにアップロードします',
        upTailscaled: 'tailscaled ファイルを NanoKVM の /usr/sbin ディレクトリにアップロードします',
        refresh: 'ページを更新します',
        notRunning: 'Tailscale はまだ実行されていません。起動操作を実行してください',
        run: '起動',
        notLogin:
          'このデバイスはまだバインドされていません。ログインしてデバイスをアカウントにバインドしてください。',
        urlPeriod: 'この URL は 10 分間有効です',
        login: 'ログイン',
        loginSuccess: 'ログイン成功',
        enable: 'Tailscale を有効化',
        deviceName: 'デバイス名',
        deviceIP: 'デバイスアドレス',
        account: 'アカウント',
        logout: 'ログアウト',
        logoutDesc: 'ログアウトしてもよろしいですか？',
        uninstall: 'Tailscale をアンインストール',
        uninstallDesc: 'Tailscale をアンインストールしてもよろしいですか？',
        okBtn: 'はい',
        cancelBtn: 'いいえ'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: '読み込み中...',
        notInstall: '未インストール',
        notConfigured: '未設定',
        stopped: '停止中',
        running: '実行中',
        connected: '接続済み',
        error: 'エラー',
        atBoot: '起動時に開始',
        notAtBoot: '起動時に開始しない',
        arguments: '引数',
        argumentsTip: '起動時にサービスへ渡すコマンドライン引数です。',
        env: '環境変数',
        envKey: '名前',
        envValue: '値',
        envAdd: '変数を追加',
        envRemove: '削除',
        configured: '設定済み',
        save: '保存',
        saved: '設定を保存しました',
        start: '起動',
        stop: '停止',
        restart: '再起動',
        logs: 'ログ',
        logsEmpty: 'ログはまだありません',
        refresh: '更新',
        binary: 'バイナリ',
        binaryShipped: 'ファームウェア同梱',
        binaryCustom: 'アップロードしたバイナリ',
        binaryUpload: 'バイナリをアップロード',
        binaryRevert: '同梱バイナリに戻す',
        binaryRevertDesc:
          'アップロードしたバイナリを削除し、ファームウェア同梱のものに戻しますか？',
        serverWarning: '制限のないサーバーはオープンプロキシになります',
        noHealthSignal:
          'このサービスはヘルス情報を出力しないため、NanoKVM はプロセスが動作していることしか確認できず、トンネルが接続済みかどうかは分かりません。',
        memoryWarning:
          '複数のリモートアクセスサービスを同時に実行するとメモリが不足する場合があります',
        resources: 'リソース',
        memory: {
          title: 'メモリ上限',
          description:
            '次回の再起動から newt の Go ヒープを {{limit}} MiB に制限します。これは newt 自身の上限であり Tailscale のものではありません。オフの場合は Go の既定値になり、GOGC=50 はどちらでも適用されます。',
          noRuntime:
            'wstunnel は Rust 製です。ガベージコレクターも設定できるヒープ上限もなく、ワーカースレッドはすでにこのデバイスの単一 CPU に合わせて動作します。',
          notApplicable: '対象外'
        },
        swap: {
          title: 'スワップファイル',
          description:
            'SD カードに 256 MB のスワップファイルを追加します。システム全体に適用され、同じスワップを Tailscale、KVM サーバー、デバイス上の他のすべてが使用します。'
        },
        okBtn: 'はい',
        cancelBtn: 'いいえ'
      },
      update: {
        title: 'アップデート',
        queryFailed: 'バージョン番号の取得に失敗しました',
        updateFailed: 'アップデートに失敗しました。もう一度お試しください。',
        isLatest: 'すでに最新バージョンです。',
        rebooting:
          '新しいカーネルをインストールして再起動しています。数分かかる場合があります。電源を切らないでください。',
        kernelUpdate:
          'このアップデートはカーネル {{version}} をインストールします。デバイスは再起動し、新しいカーネルが起動しない場合は自動的に現在のカーネルに戻ります。',
        rolledBack:
          'カーネル {{version}} が起動しなかったため、デバイスは以前のカーネルに戻りました。',
        available: '新しいバージョンが利用可能です。アップデートしてもよろしいですか？',
        updating: 'アップデート中、お待ちください...',
        confirm: 'はい',
        cancel: 'いいえ',
        preview: 'プレビューアップデート',
        previewDesc: '新機能や改善をいち早く体験する',
        previewTip:
          'プレビューアップデートには不安定な部分や不完全な機能が含まれる場合があります！',
        customServer: {
          title: 'カスタム更新サーバー',
          desc: '指定したサーバーでオンラインアップデートを確認し、ダウンロードします',
          invalidUrl:
            'クエリ、フラグメント、latest.json を含まない、有効な HTTP または HTTPS のサーバーディレクトリを入力してください。',
          loadFailed: '更新サーバーの設定を読み込めませんでした。',
          saveFailed: '更新サーバーの設定を保存できませんでした。',
          saved: '更新サーバーの設定を保存しました。',
          save: '保存',
          confirmTitle: 'カスタム更新サーバーを使用しますか？',
          confirmDesc:
            'SHA-512 で確認できるのは、パッケージがこのサーバーから提供されたマニフェストと一致することだけです。そのパッケージが NanoKVM の公式リリースであることは保証されません。不具合のあるサーバーや悪意のあるサーバーを使用すると、デバイスが使用不能になったり、データが失われたり、システムが侵害されたりする可能性があります。',
          confirm: 'そのまま使用',
          previewDisabled:
            'カスタム更新サーバーが有効な間は、プレビュー版アップデートを利用できません。'
        },
        offline: {
          kernelNotice:
            'このパッケージにはカーネルが含まれます。予備スロットに書き込んで再起動し試験起動します。起動しなかった場合は自動的に現在のカーネルへ戻ります。',
          kernelConfirm: 'カーネルをインストール',
          kernelCancel: 'キャンセル',
          title: 'オフラインアップデート',
          desc: 'ローカルインストールパッケージでアップデートする',
          upload: 'アップロード',
          checksumPlaceholder: 'SHA-256チェックサム（任意）',
          invalidChecksum: 'SHA-256チェックサムは64文字の16進数である必要があります。',
          checksumMismatch:
            'SHA-256の検証に失敗しました。パッケージが破損している可能性があります。',
          invalidName:
            'ファイル名の形式が正しくありません。GitHub リリースページにアクセスしてインストールパッケージをダウンロードしてください。',
          updateFailed: 'アップデートに失敗しました。もう一度お試しください。'
        }
      },
      account: {
        title: 'アカウント',
        webAccount: 'ウェブアカウント名',
        role: 'ロール',
        roles: {
          admin: '管理者',
          user: 'ユーザー'
        },
        password: 'パスワード',
        updateBtn: '変更',
        logoutBtn: 'ログアウト',
        logoutDesc: 'ログアウトしてもよろしいですか？',
        okBtn: 'はい',
        cancelBtn: 'いいえ',
        users: {
          title: 'ユーザー',
          create: 'ユーザーを作成',
          enabled: '有効',
          disabled: '無効',
          deviceOwner: 'デバイスの所有者',
          resetPassword: 'パスワードをリセット',
          delete: '削除',
          deleteConfirm: 'このユーザーを削除し、そのセッションをすべて無効にしますか？',
          created: 'ユーザーを作成しました',
          deleted: 'ユーザーを削除しました',
          passwordUpdated: 'パスワードを更新しました',
          loadFailed: 'ユーザーの取得に失敗しました',
          saveFailed: 'ユーザーの保存に失敗しました',
          deleteFailed: 'ユーザーの削除に失敗しました'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw アシスタント',
      empty: 'パネルを開いてタスクを開始して開始します。',
      inputPlaceholder: 'PicoClaw に実行してほしいことを説明してください',
      newConversation: '新しい会話',
      processing: '処理中...',
      agent: {
        defaultTitle: '一般アシスタント',
        defaultDescription: '一般的なチャット、検索、およびワークスペースのヘルプ。',
        kvmTitle: 'リモート操作',
        kvmDescription: 'NanoKVM を通じてリモート ホストを操作します。',
        switched: 'エージェントの役割が切り替わりました',
        switchFailed: 'エージェントの役割を切り替えることができませんでした'
      },
      send: '送信',
      cancel: 'キャンセル',
      status: {
        connecting: 'ゲートウェイに接続しています...',
        connected: 'PicoClaw セッションが接続されました',
        disconnected: 'PicoClaw セッションが終了しました',
        stopped: '停止要求が送信されました',
        runtimeStarted: 'PicoClaw ランタイムが開始されました',
        runtimeStartFailed: 'PicoClaw ランタイムの開始に失敗しました',
        runtimeStopped: 'PicoClaw ランタイムが停止しました',
        runtimeStopFailed: 'PicoClaw ランタイムの停止に失敗しました',
        controlSwitchedToMCP: '制御が外部 MCP サービスに切り替わりました'
      },
      connection: {
        runtime: {
          checking: 'チェック中',
          restoring: 'Restoring PicoClaw',
          ready: 'ランタイムの準備が完了しました',
          stopped: 'ランタイムが停止しました',
          blockedByMCP: '外部 MCP 制御が有効です',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'ランタイムが使用できません',
          configError: '構成エラー'
        },
        transport: {
          connecting: '接続中',
          connected: '接続されました',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'アイドル状態',
          busy: '忙しいです'
        }
      },
      message: {
        toolAction: 'アクション',
        observation: '観察',
        screenshot: 'スクリーンショット'
      },
      overlay: {
        locked: 'PicoClaw がデバイスを制御しています。手動入力が一時停止されます。'
      },
      control: {
        picoclaw: 'デバイス制御: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'デバイス制御: 外部 MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'デバイス制御: オフ',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: '制御を付与',
        release: '解除',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw 制御を付与しました',
        released: 'PicoClaw 制御を解除しました',
        grantFailed: 'PicoClaw 制御の付与に失敗しました',
        releaseFailed: 'PicoClaw 制御の解除に失敗しました',
        grantConfirmTitle: 'デバイス制御を PicoClaw に切り替えますか?',
        grantConfirmDesc: '外部 MCP のデバイス書き込みは中断されます。'
      },
      install: {
        install: 'PicoClaw をインストールする',
        installing: 'PicoClaw をインストールしています',
        success: 'PicoClaw は正常にインストールされました',
        failed: 'PicoClaw のインストールに失敗しました',
        uninstalling: 'ランタイムをアンインストールしています...',
        uninstalled: 'ランタイムは正常にアンインストールされました。',
        uninstallFailed: 'アンインストールに失敗しました。',
        requiredTitle: 'PicoClaw がインストールされていません',
        requiredDescription:
          'PicoClaw ランタイムを開始する前に PicoClaw をインストールしてください。',
        progressDescription: 'PicoClaw をダウンロードしてインストールしています。',
        stages: {
          preparing: '準備中',
          downloading: 'ダウンロード中',
          extracting: '展開中',
          verifying: '検証中',
          installing: 'インストール中',
          installed: 'インストール完了',
          install_timeout: 'タイムアウト',
          install_failed: '失敗'
        }
      },
      model: {
        requiredTitle: 'モデル構成が必要です',
        requiredDescription: 'PicoClaw チャットを使用する前に、PicoClaw モデルを構成します。',
        docsTitle: '構成ガイド',
        docsDesc: 'サポートされているモデルとプロトコル',
        menuLabel: 'モデルの構成',
        modelIdentifier: 'モデル識別子',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API キー',
        apiKeyPlaceholder: 'モデルの API キーを入力してください',
        save: '保存',
        saving: '保存中',
        saved: 'モデル構成が保存されました',
        saveFailed: 'モデル構成の保存に失敗しました',
        invalid: 'モデル識別子、API Base URL、API キーは必須です'
      },
      uninstall: {
        menuLabel: 'アンインストール',
        confirmTitle: 'PicoClaw のアンインストール',
        confirmContent:
          'PicoClaw をアンインストールしてもよろしいですか?これにより、実行可能ファイルとすべての構成ファイルが削除されます。',
        confirmOk: 'アンインストール',
        confirmCancel: 'キャンセル'
      },
      history: {
        title: '履歴',
        loading: 'セッションを読み込み中...',
        emptyTitle: '履歴はまだありません',
        emptyDescription: '以前の PicoClaw セッションがここに表示されます。',
        loadFailed: 'セッション履歴のロードに失敗しました',
        deleteFailed: 'セッションの削除に失敗しました',
        deleteConfirmTitle: 'セッションを削除します',
        deleteConfirmContent: '「{{title}}」を削除してもよろしいですか?',
        deleteConfirmOk: '削除',
        deleteConfirmCancel: 'キャンセル',
        messageCount_one: '{{count}} メッセージ',
        messageCount_other: '{{count}} メッセージ',
        messageCount: '{{count}} メッセージ'
      },
      config: {
        startRuntime: 'PicoClaw を開始',
        stopRuntime: 'PicoClaw を停止'
      },
      start: {
        enableConfirmTitle: '制御を PicoClaw に切り替えますか？',
        enableConfirmDesc: 'PicoClaw を開始すると外部 MCP サービスが無効になります。',
        enableConfirmOk: 'PicoClaw を開始',
        enableConfirmCancel: 'キャンセル',
        title: 'PicoClaw を開始',
        description: 'ランタイムを起動して、PicoClaw アシスタントの使用を開始します。',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'エラーが発生しました',
      refresh: '更新',
      panel: 'このパネルは停止しました',
      retry: '再試行',
      reload: 'ページを再読み込み'
    },
    fullscreen: {
      toggle: '全画面表示切り替え'
    },
    menu: {
      collapse: 'メニューを折りたたむ',
      expand: 'メニューを展開する'
    }
  }
};

export default ja;
