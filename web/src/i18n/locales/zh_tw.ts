const zh_tw = {
  translation: {
    head: {
      desktop: '遠端桌面',
      login: '登入',
      changePassword: '更改密碼',
      terminal: '終端機',
      wifi: 'Wi-Fi',
      storage: '映像檔掛載 / 檔案共享'
    },
    auth: {
      login: '登入',
      placeholderUsername: '使用者名稱',
      placeholderPassword: '密碼',
      placeholderPassword2: '請再次輸入密碼',
      noEmptyUsername: '使用者名稱不能為空',
      noEmptyPassword: '密碼不能為空',
      noAccount: '找不到使用者，請重新整理網頁或重設密碼',
      invalidUser: '使用者名稱或密碼錯誤',
      locked: '登入次數過多，請稍後重試',
      globalLocked: '系統受保護，請稍後重試',
      error: '非預期性錯誤',
      changePassword: '更改密碼',
      changePasswordDesc: '為了您的裝置安全，請修改登入密碼。',
      differentPassword: '密碼不一致',
      illegalUsername: '使用者名稱包含非法字元',
      illegalPassword: '密碼包含非法字元',
      forgetPassword: '忘記密碼',
      ok: '確定',
      cancel: '取消',
      loginButtonText: '登入',
      tips: {
        reset1: '長按 NanoKVM 上的 BOOT 按鍵 10 秒鐘來重設帳號。',
        reset2: '詳細操作方法可參閱本文件：',
        reset3: '網頁預設帳號：',
        reset4: 'SSH 預設帳號：',
        change1: '請注意，此操作將同時更新下列密碼：',
        change2: '網頁登入密碼',
        change3: 'root user 密碼 (SSH登入密碼)',
        change4: '如果您忘記密碼，需要長按 NanoKVM 上的 BOOT 按鍵來重設密碼。'
      },
      placeholderCurrentPassword: '請輸入目前密碼',
      passwordLength: '密碼長度必須為 8 到 72 個字元',
      invalidCurrentPassword: '目前密碼錯誤'
    },
    wifi: {
      title: 'Wi-Fi',
      description: '設定 NanoKVM Wi-Fi',
      success: '請檢查 NanoKVM 的網路狀態，並存取新的 IP 位址。',
      failed: '操作失敗，請重試。',
      invalidMode: '目前模式不支援設定網路。請先前往裝置啟用 Wi-Fi 配置模式。',
      confirmBtn: '確定',
      finishBtn: '完成',
      ap: {
        authTitle: '需要身份驗證',
        authDescription: '請輸入 AP 密碼繼續',
        authFailed: 'AP 密碼無效',
        passPlaceholder: 'AP 密碼',
        verifyBtn: '驗證'
      }
    },
    desktop: {
      readOnly: '唯讀',
      readOnlyTip:
        '此帳號沒有鍵盤與滑鼠操作權限，無法向目標機發送鍵盤與滑鼠事件，主控台僅顯示畫面。'
    },
    noConsole: {
      title: '此帳戶沒有控制台權限',
      description: '此帳戶被授予了其他功能。可以開啟下方可用頁面，或聯絡管理員授予控制台權限。',
      nothing: '此帳戶還沒有任何功能權限，請聯絡管理員授予。'
    },
    screen: {
      scale: '缩放',
      title: '螢幕',
      video: '編碼格式',
      videoDirectTips: '本模式需先啟用 HTTPS，請前往「設定 -> 設備」中開啟',
      resolution: '解析度',
      controlRegion: {
        title: '滑鼠校正',
        description:
          '當受控裝置使用非 16:9 解析度，且游標在水平或垂直方向出現偏移時，請使用此設定。',
        off: '關閉',
        auto: '自動',
        autoWarning: '當使用者應用程式使用全黑背景時，校正可能會失敗。',
        manual: '手動',
        selectedResolution: '框選解析度',
        unused: '未使用',
        originalResolution: '原始解析度',
        selectResolution: '選擇原始解析度',
        addResolution: '新增自訂解析度',
        add: '新增',
        duplicateResolution: '此解析度已存在。',
        width: '寬度',
        height: '高度',
        apply: '計算並套用',
        invalidResolution: '請在影片準備完成後輸入有效的原始解析度。',
        select: '選擇區域',
        clear: '恢復自動偵測',
        saveFailed: '無法儲存輸入區域。',
        tooSmall: '所選區域太小。',
        previewUnavailable: '無法預覽',
        clearConfirm: '要恢復自動黑邊偵測嗎？',
        dragHint: '拖曳以選擇遠端桌面區域',
        finish: '完成',
        confirm: '確認',
        cancel: '取消'
      },
      auto: '自動',
      autoTips:
        '在某些特定解析度下可能會出現畫面撕裂或滑鼠偏移的情況。請調整遠端主機的解析度或停用自動模式。',
      fps: '更新頻率',
      customizeFps: '自定義',
      quality: '品質',
      qualityLossless: '無損',
      qualityHigh: '高',
      qualityMedium: '中',
      qualityLow: '低',
      frameDetect: '影格檢測',
      frameDetectTip: '計算影格之間的差異。當遠端主機畫面未偵測到任何變更時，停止視訊傳輸串流。',
      resetHdmi: '重置 HDMI',
      mixedH264: {
        title: 'H.264 串流衝突',
        description:
          '偵測到 H.264 Direct 和 H.264 WebRTC 同時使用，可能導致畫面撕裂或影片損壞。請只保留一種 H.264 模式。'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC 連線失敗',
        description: '請檢查網路連線或切換影片模式。'
      },
      captureStatus: {
        hdmiError: 'HDMI 畫面異常',
        unsupportedResolution: '目前解析度不支援',
        retrieving: '正在取得畫面...',
        changingResolution: '正在切換解析度...',
        updateFailed: '畫面暫時無法更新',
        videoError: '影片顯示異常',
        noHdmi: '未偵測到 HDMI 訊號',
        unavailable: '畫面暫時無法顯示'
      }
    },
    keyboard: {
      title: '鍵盤',
      paste: '貼上',
      tips: '僅支援標準鍵盤的字母和符號',
      placeholder: '請輸入內容',
      submit: '送出',
      virtual: '虛擬鍵盤',
      readClipboard: '從剪貼簿讀取',
      clipboardPermissionDenied: '剪貼簿權限被拒絕。請允許您的瀏覽器存取剪貼簿。',
      clipboardReadError: '無法讀取剪貼簿',
      dropdownEnglish: '英語',
      dropdownGerman: '德語',
      dropdownFrench: '法語',
      dropdownRussian: '俄語',
      shortcut: {
        title: '快捷鍵',
        custom: '自定義',
        capture: '點選此處錄製快捷鍵',
        clear: '清空',
        save: '儲存',
        captureTips: '擷取系統級按鍵（如 Windows 鍵）需要全螢幕權限。',
        enterFullScreen: '切換全螢幕模式。'
      },
      leaderKey: {
        title: '引導鍵',
        desc: '繞過瀏覽器限制並將系統捷徑直接傳送到遠端主機。',
        howToUse: '如何使用',
        simultaneous: {
          title: '同步模式',
          desc1: '按住引導鍵不放，同時按下目標快捷鍵。',
          desc2: '直觀，但可能與系統快速鍵衝突。'
        },
        sequential: {
          title: '順序模式',
          desc1: '點擊引導鍵開始 → 依序點擊快捷鍵 → 再次點擊引導鍵結束。',
          desc2: '需要更多步驟，但完全避免了系統衝突。'
        },
        enable: '啟用引導鍵',
        tip: '設為引導鍵後，該按鍵將僅用於觸發快捷鍵，不再作為普通按鍵使用。',
        placeholder: '請按下引導鍵',
        shiftRight: '右 Shift',
        ctrlRight: '右 Ctrl',
        metaRight: '右 Win',
        submit: '送出',
        recorder: {
          rec: '記錄',
          activate: '啟用按鍵',
          input: '請按快捷鍵...'
        }
      }
    },
    mouse: {
      title: '滑鼠',
      cursor: '游標樣式',
      default: '預設游標',
      pointer: '懸浮游標',
      cell: '儲存格游標',
      text: '文字游標',
      grab: '抓取游標',
      hide: '隱藏游標',
      mode: '滑鼠模式',
      absolute: '絕對模式',
      relative: '相對模式',
      direction: '滾輪方向',
      scrollUp: '向上',
      scrollDown: '向下',
      speed: '滾輪速度',
      fast: '快',
      slow: '慢',
      requestPointer: '正在使用滑鼠相對模式。請按一下桌面以取得滑鼠游標。',
      resetHid: '重設 HID',
      hidOnly: {
        title: 'HID-Only 模式',
        desc: '如果您的滑鼠和鍵盤沒有反應，且重設 HID 無效，可能是 NanoKVM 與您的裝置間有相容性問題。請嘗試啟用 HID-Only 模式以獲得更好的相容性。',
        tip1: '啟用 HID-Only 模式將會停用虛擬隨身碟和虛擬網卡的功能',
        tip2: '在 HID-Only 模式下，映像檔掛載功能將被停用',
        tip3: 'NanoKVM 將在切換模式後自動重新啟動',
        enable: '啟用 HID-Only 模式',
        disable: '停用 HID-Only 模式'
      }
    },
    storage: {
      title: '映像檔掛載 / 檔案共享',
      empty: '還沒有映像檔。可以上傳 ISO/IMG，也可以讓受控裝置拷到這張磁碟上。',
      notAttached: '尚未裝載任何媒體',
      attached: '已裝載：{{name}}',
      eject: '彈出',
      mount: '裝載',
      mountMode: '裝載模式',
      mountModeDisk: '磁碟模式',
      mountModeCdrom: '光碟模式',
      openManager: '開啟檔案管理員',
      refresh: '重新整理',
      delete: '刪除',
      deleteTitle: '刪除媒體',
      deleteConfirm: '確定刪除 {{name}}？裡面的檔案會一起遺失。',
      downloadImage: '下載映像檔',
      typePartition: '共享檔案',
      volumeLabel: '可移除磁碟',
      groups: {
        share: '檔案共享',
        images: '映像檔'
      },
      failed: '操作失敗',
      hidOnly: 'HID-Only 模式：虛擬隨身碟無法使用',
      disabled: '虛擬隨身碟未啟用（設定 - 設備）',
      force: {
        title: '強制收回媒體？',
        desc: '受控裝置還掛著這個媒體。強制收回會讓 USB 裝置重新列舉，它還沒寫入磁碟的資料可能會遺失。',
        menuHint: '受控裝置正在使用該媒體，請到檔案管理員頁面強制收回。'
      },
      tips: {
        title: '如何上傳',
        usb1: '透過 USB 將 NanoKVM 連接到您的電腦。',
        usb2: '在本頁面以大容量儲存模式掛載共享檔案；',
        usb3: '開啟電腦上的虛擬磁碟，將映象檔案複製到虛擬磁碟的根目錄下。',
        usb4: '複製完成後退出磁碟，映像檔隨後會出現在清單中。',
        scp1: '確保 NanoKVM 和您的電腦位於同一區域網路。',
        scp2: '開啟電腦上的終端機，使用 SCP 指令將映像檔案上傳到 NanoKVM 的 /data 目錄下。',
        scp3: '範例：scp your-image-path root@your-nanokvm-ip:/data',
        tfCard: 'microSD 卡',
        tf1: '此方法適用於 Linux 系統',
        tf2: '從 NanoKVM 拔出 microSD 卡（FULL 版本請先拆開外殼）。',
        tf3: '使用 USB 讀卡機將 microSD 卡連接至電腦。',
        tf4: '將映像檔複製到 microSD 卡的 /data 目錄下。',
        tf5: '將 microSD 卡重新插回 NanoKVM。'
      },
      job: {
        running: '正在處理 {{name}}'
      },
      enableDisk: {
        title: '啟用虛擬隨身碟？',
        desc: '裝載媒體需要虛擬隨身碟，目前它是關閉的。現在啟用？受控裝置的鍵盤與滑鼠會短暫中斷。',
        menuHint: '虛擬隨身碟未啟用，請到 設定 - 設備 開啟。',
        askDevice: '虛擬磁碟目前已關閉，需要擁有「設備設定」權限的帳號先將其開啟。'
      },
      add: {
        title: '新增媒體',
        upload: '上傳 ISO / IMG',
        fromUrl: '從 URL 下載',
        url: '映像檔網址',
        urlRequired: '請輸入映像檔網址',
        onlyImages: '只能掛載 .iso 和 .img 檔案',
        dropHint: '把 ISO / IMG 拖到這裡，或點擊選擇檔案',
        cdromOnlyIso: '光碟模式只能掛載 ISO 光碟映像檔',
        uploadFailed: '上傳失敗',
        downloadFailed: '下載失敗',
        uploading: '正在上傳 {{name}}',
        downloading: '正在下載 {{name}}',
        done: '{{name}} 已就緒',
        checksumFailed: '映像檔與 SHA-256 校驗值不符。',
        blocked: '受控裝置正在使用這張磁碟',
        copyHint: '也可以透過 SCP 把映像檔放進資料分割區，或在受控裝置上直接複製。'
      },
      files: {
        selectMedium: '選擇一個媒體，查看裡面的檔案。',
        reasonAttached: '該媒體正裝載給受控裝置使用。',
        reasonAttachedHint: '請先彈出：受控裝置必須釋放這張磁碟，裡面的檔案才能讀取。',
        reasonNotBrowsable: '這個媒體裡的檔案系統無法在網頁端開啟。',
        reasonNoLoop: '本機不支援掛載磁碟映像檔（核心未啟用 loop）。',
        reasonISOImage: 'ISO 光碟映像檔不支援在網頁端瀏覽目錄，映像檔本身仍可正常掛載給受控裝置',
        empty: '這個資料夾是空的',
        noMatch: '沒有符合的檔案',
        free: '剩餘 {{size}}',
        search: '搜尋目前資料夾',
        sort: {
          name: '名稱',
          size: '大小',
          type: '類型',
          asc: '升冪',
          desc: '降冪'
        },
        upload: '上傳檔案到媒體',
        deleteFolderTitle: '刪除資料夾',
        deleteFolderConfirm: '刪除資料夾 {{name}} 及其中的所有內容？此操作無法復原。',
        uploading: '正在上傳…',
        uploadFailed: '上傳失敗',
        download: '下載',
        delete: '刪除',
        deleteTitle: '刪除檔案',
        deleteConfirm: '確定從媒體中刪除 {{name}}？'
      },
      yes: '確定',
      no: '取消'
    },
    script: {
      title: '指令碼',
      upload: '上傳',
      run: '執行',
      runBackground: '背景執行',
      runFailed: '執行失敗',
      attention: '注意',
      delDesc: '確定要刪除該檔案嗎？',
      confirm: '確定',
      cancel: '取消',
      delete: '刪除',
      close: '關閉'
    },
    terminal: {
      title: '終端機',
      nanokvm: 'NanoKVM 終端機',
      serial: 'Serial Port 終端機',
      serialPort: '序列埠',
      serialPortPlaceholder: '請輸入 Serial Port',
      baudrate: '鮑率',
      parity: '同位檢查',
      parityNone: 'None',
      parityEven: '偶同位',
      parityOdd: '奇同位',
      flowControl: '流量控制',
      flowControlNone: 'None',
      flowControlSoft: '軟體',
      flowControlHard: '硬體',
      dataBits: '資料位元',
      stopBits: '停止位元',
      confirm: '確定'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: '發送指令中...',
      sent: '指令已發送',
      input: '請輸入 MAC 位址',
      ok: '確定'
    },
    download: {
      disabled:
        '共享資料夾正被受控裝置使用，/data 目前唯讀：無法寫入新映像檔，既有檔案仍可正常下載。',
      sha256: 'SHA-256（可選）',
      sha256Placeholder: '請輸入 64 位元 SHA-256 校驗和',
      invalidSHA256: 'SHA-256 必須是 64 位元十六進位字串',
      cancel: '取消',
      cancelFailed: '取消下載失敗'
    },
    power: {
      title: '電源控制',
      showConfirm: '顯示確認框',
      showConfirmTip: '電源操作需要二次確認',
      reset: '重新啟動',
      power: '電源',
      powerShort: '電源 (短按)',
      powerLong: '電源 (長按)',
      resetConfirm: '確認執行重新啟動的按鍵操作嗎？',
      powerConfirm: '確認執行電源的按鍵操作嗎？',
      okBtn: '確認',
      cancelBtn: '取消'
    },
    devices: {
      title: '設備',
      stale: '無法取得設備的即時狀態，正在重新連線。',
      empty: '尚未設定攝影機、麥克風或喇叭插槽。請在「設定 - 裝置」中新增。',
      available: '可用',
      waiting: '主機正在等待來源',
      hostOpen: '主機已開啟',
      hostIdle: '主機閒置',
      hostPlaying: '主機正在播放音訊',
      hostSending: '主機播放中',
      sending: '正從本瀏覽器傳送',
      receiving: '正在本瀏覽器播放',
      black: '黑畫面',
      silence: '數位靜音',
      resuming: '等待恢復',
      stop: '停止分享',
      stopListening: '停止聆聽',
      disconnect: '中斷連線',
      takeover: '接管',
      refused: '{{owner}} 正透過 {{source}} 使用',
      connectedSources_one: '已連線 {{count}} 個來源',
      connectedSources_other: '已連線 {{count}} 個來源',
      connectedSources: '已連線 {{count}} 個來源',
      connection: {
        connecting: '連線中',
        connected: '即時',
        disconnected: '重新連線中'
      },
      share: {
        camera: '分享攝影機',
        microphone: '分享麥克風',
        speaker: '聆聽',
        usbDevice: '分享 USB'
      },
      permission: {
        denied: '已在瀏覽器的網站設定中被封鎖',
        prompt: '瀏覽器將要求存取權限',
        insecure:
          '本頁面未透過 HTTPS 提供，瀏覽器因此停用了該裝置。請在「設定 - 網路」中啟用 HTTPS。'
      },
      capture: {
        unsupported: '此瀏覽器無法擷取音訊或影像',
        camera: '此瀏覽器無法編碼攝影機畫面',
        microphone: '此瀏覽器無法處理麥克風音訊',
        speaker: '此瀏覽器無法播放音訊'
      },
      mic: {
        mute: '靜音',
        unmute: '取消靜音'
      },
      revoked: {
        released: '分享已停止',
        lease_expired: '租約在本瀏覽器回來之前就已到期',
        admin_disconnect: '管理員中斷了所有來源',
        slot_removed: '此插槽已被移除',
        slot_changed: '此插槽已重新設定',
        taken_over: '管理員接管了此插槽'
      },
      usb: {
        surrendered: 'USB 直通正佔用鍵盤與滑鼠',
        surrenderedDesc:
          '遠端主機看到的是匯入的裝置，而不是 NanoKVM 的鍵盤、滑鼠與虛擬媒體。工作階段結束後它們會自行恢復。',
        unsupported: 'WebUSB 需要 Chromium 核心瀏覽器',
        insecure:
          '本頁面未透過 HTTPS 提供，瀏覽器因此停用了 WebUSB。請在「設定 - 網路」中啟用 HTTPS。',
        session: '正在直通 {{device}}（{{mode}}）',
        idle: '沒有直通工作階段',
        mode: {
          hybrid: '混合',
          exact: '精確'
        }
      }
    },
    settings: {
      title: '設定',
      display: {
        title: '顯示',
        loading: '載入中...',
        active: '目前 EDID',
        activeUnknown: 'NanoKVM 自啟動以來未寫入過 EDID，因此主機看到的顯示器識別資訊未知。',
        appliedAt: '套用於 {{time}}',
        download: '下載',
        downloadBackup: '下載上一個',
        preset: '顯示器預設',
        presetPlaceholder: '選擇顯示器',
        upload: '上傳',
        selected: '已選擇的 EDID',
        errors: '錯誤',
        warnings: '警告',
        info: '資訊',
        unknownMonitor: '未知顯示器',
        edidVersion: 'EDID {{version}}',
        audioYes: '支援音訊',
        audioNo: '無音訊',
        extensionBlocks: '擴充區塊：{{blocks}}',
        apply: '套用',
        applyTitle: '要套用此 EDID 嗎？',
        before: '目前',
        after: '新的',
        hdmiNotice: '寫入 EDID 期間視訊擷取會中斷，完成後會自行恢復。',
        powerCycleNotice: '必須將本裝置實體斷電後重新接上電源，新的 EDID 才會生效。',
        powerCycleUnverified:
          '寫入未通過驗證，因此在將本裝置實體斷電並重新接上電源之前，視訊晶片會一直保留它目前的內容。',
        applied: 'EDID 已套用並驗證通過。',
        applyFailed: 'EDID 套用失敗。',
        busy: '視訊晶片忙碌中，請重試。',
        unsupported: '本裝置不支援變更 EDID。',
        toolMissing: '此韌體中缺少 EDID 工具。',
        noAudio: '此 EDID 未宣告音訊，主機可能會停止輸出聲音。',
        oldVersion: '此 EDID 使用低於 1.4 的版本。',
        interlaced: '偏好時序為交錯掃描。',
        tooLarge: '偏好時序高於 1920x1080 60 Hz，超出 NanoKVM 的擷取能力。',
        recovery: '復原',
        recoveryNeeded:
          '上一次寫入未通過驗證，視訊晶片的 EDID 區域處於未知狀態。請復原出廠 EDID，使其回到已知狀態。',
        recoveryDesc: '當套用的 EDID 導致主機沒有畫面時，將已知的 EDID 重新寫回視訊晶片。',
        restoreFactory: '復原出廠 EDID',
        restoreBackup: '復原上一個 EDID',
        restoreTitle: '要復原此 EDID 嗎？',
        restoreFactoryTarget: 'NanoKVM 出廠隨附的 EDID。',
        restoreBackupTarget: '最近一次備份，套用於 {{time}}。',
        restoreNotice: '復原與套用採用相同的寫入方式，後果也相同。',
        restored: 'EDID 已復原並通過驗證。',
        restoreFailed: 'EDID 復原失敗。',
        mismatchTitle: '寫入內容與讀回內容',
        mismatchWritten: '已寫入',
        mismatchRead: '讀回',
        restoreOkBtn: '復原',
        hardware: '偵測到的硬體：{{hardware}}',
        hardwareUnknown: '未知',
        confirmWord: '套用',
        confirmPrompt: '輸入 {{word}} 以啟用套用按鈕。',
        okBtn: '套用',
        cancelBtn: '取消'
      },
      presentation: {
        title: 'USB 呈現',
        loading: '載入中...',
        current: '目前的 USB 呈現',
        noProfile: '未套用任何設定檔',
        linked: '已連結的功能',
        hostState: '主機 USB',
        hostUnbound: '控制器未繫結',
        hdmiState: 'HDMI 輸入',
        hdmiSignal: '有訊號',
        hdmiUnreported: '尚無擷取回報',
        endpoints: '端點',
        fifos: 'FIFO 插槽',
        pending: '待處理的變更',
        pendingEdits: '未儲存的識別資訊修改',
        pendingProfile: '已選取 {{profile}}，但尚未套用',
        pendingNone: '無',
        lastApply: '上次套用',
        applyFailed: '{{time}} 在 {{profile}} 上失敗',
        applyClean: '沒有失敗紀錄',
        lastKnownGood: '上次已知可用',
        rollbackTarget: '回復目標',
        rollbackNone: '無',
        powerCyclePending: '控制器已從主機收回。請將所連接的電腦斷電後重新開機，以取回該裝置。',
        rollback: '回復',
        rollbackTitle: '回復到 {{profile}}？',
        rollbackDesc: '小工具將重新列舉，USB 功能會短暫中斷。',
        profile: 'USB 設定檔',
        builtIn: '內建',
        descriptors: '描述元',
        imported: '已匯入',
        clone: '複製',
        cloneTitle: '複製此設定檔',
        cloneToEdit: '內建設定檔維持唯讀。複製此設定檔後即可編輯其身分。',
        profileName: '設定檔名稱',
        profileNameHint: '可使用小寫字母、數字、點、底線與連字號。',
        import: '匯入套件',
        export: '匯出套件',
        delete: '刪除',
        deleteTitle: '要刪除此設定檔嗎？',
        deleteDesc: '從 NanoKVM 刪除 {{profile}}。',
        identity: 'USB 身分',
        preset: '預設身分',
        presetPlaceholder: '從已知設備複製身分',
        presetHint: '預設會填入 Vendor ID、Product ID 與兩個名稱欄位，但不會帶來描述元。',
        presetSource: '{{source}} 中記錄的身分',
        vendorId: 'Vendor ID',
        foreignVendor: '此 Vendor ID 屬於另一家製造商',
        productId: 'Product ID',
        bcdUSB: 'USB 版本',
        bcdDevice: '設備版本',
        manufacturer: '製造商',
        product: '產品',
        serial: '序號',
        configuration: '設定字串',
        hidLayout: 'HID 裝置',
        hidRoleKeyboard: '鍵盤',
        hidRoleRelative: '滑鼠（相對）',
        hidRoleAbsolute: '指標（絕對）',
        hidOff: '不存在',
        hidInterface: '介面 {{index}}',
        hidBootKeyboardShared:
          '鍵盤與其他功能共用一個介面，因此不再提供 boot 通訊協定回報。部分 BIOS 與 UEFI 將無法辨識它。',
        functions: '功能',
        descriptorAssets: '已儲存的描述元檔案：{{count}}',
        endpointUse: 'IN 已用 {{inUse}}、閒置 {{inFree}}；OUT 已用 {{outUse}}、閒置 {{outFree}}',
        apply: '套用',
        applyTitle: '要套用此 USB 設定檔嗎？',
        applyDesc: 'NanoKVM 將向已連線的電腦呈現 {{profile}}。',
        reconnect: '在小工具重新繫結期間，鍵盤、滑鼠與其他 USB 功能會短暫中斷。',
        applyLinks: '連結：{{functions}}',
        applyRemoves: '移除：{{functions}}',
        applyNoHid: '本次套用後不會保留任何 HID 功能，鍵盤與滑鼠將停止運作。',
        applyRollback: '套用失敗會回到 {{profile}}。',
        recoveryPowerCycle:
          '本次套用不會保留任何 HID，因此主機若失去回應，只能以斷電重開的方式復原。',
        recoveryReboot: '複合裝置中會少掉一個介面，主機可能需要重新開機才能重新繫結其餘介面。',
        recoveryHdmiReset: '視訊功能會被重新建立，因此其背後的擷取流程也會重設。',
        recoveryReconnect: '主機將重新列舉此裝置，USB 功能會短暫中斷。',
        cancel: '取消',
        noFunctions: '沒有已連結的功能',
        loadFailed: '載入呈現設定檔失敗',
        operationFailed: 'USB 呈現操作失敗'
      },
      passthrough: {
        title: 'USB 直通',
        loading: '載入中...',
        mode: '模式',
        hybrid: '混合',
        exact: '完全',
        hybridDesc: '為相容裝置保留 boot 鍵盤與相對滑鼠。',
        exactDesc: '以直通裝置取代 NanoKVM 的每一項 USB 功能。',
        hybridWarning: '混合模式仍可使用鍵盤與相對滑鼠',
        hybridWarningDesc: '直通功能啟用期間，儲存、USB 網路與絕對指標會中斷。',
        hidWarning: '啟動直通會讓出鍵盤、滑鼠與虛擬媒體',
        hidWarningDesc:
          'NanoKVM 只有一個 USB 裝置控制器，而代理需要獨佔它。因此工作階段執行期間，遠端主機看到的是被直通的裝置，而不是 NanoKVM 的鍵盤、滑鼠與虛擬媒體。工作階段一停止，它們就會自動恢復。此網頁介面不受影響，您隨時可以在本頁停止工作階段。',
        hidWarningSafeDesc:
          'NanoKVM 只有一個 USB 裝置控制器，而代理需要獨佔它。因此工作階段執行期間，遠端主機看到的是被直通的裝置，而不是 NanoKVM 的鍵盤、滑鼠與虛擬媒體。工作階段停止後它們就會恢復。',
        isoLabel: '允許等時傳輸',
        isoHint: '放行網路攝影機、麥克風等串流裝置。沒有人測過此硬體能跑到多少。',
        isoWarning: '等時傳輸在此尚未驗證，可能占住鍵盤與滑鼠直到你停止工作階段',
        info: {
          title: '說明',
          hybrid:
            '混合模式仍可使用鍵盤與相對滑鼠。直通裝置啟用期間，儲存、USB 網路與絕對指標會中斷。',
          exact:
            '完全模式以直通裝置取代 NanoKVM 的每一項 USB 功能。工作階段停止後，鍵盤、滑鼠與虛擬媒體會自動恢復。',
          udc: 'NanoKVM 只有一個 USB 裝置控制器，而代理需要獨佔它，所以工作階段執行期間上述功能都會消失。',
          web: '此網頁介面不受影響，您隨時可以在本頁停止工作階段。',
          network:
            '請透過乙太網路或 Wi-Fi 啟動直通。從 NanoKVM 的 USB 網路啟動會被拒絕，因為該連線屆時會消失。',
          iso: '在允許等時傳輸之前，網路攝影機、麥克風等等時傳輸裝置會被拒絕。該路徑可用，但從未在此硬體上測量過，因此其吞吐量應視為未知。',
          camera: '設備選單中瀏覽器的攝影機與麥克風，仍是給遠端主機提供它們的成熟方式。'
        },
        session: '工作階段',
        activeDesc: '已匯入一台裝置，代理正占用 USB 控制器。',
        inactiveDesc: '目前沒有工作階段。鍵盤、滑鼠與虛擬媒體運作正常。',
        device: '裝置',
        busId: '匯流排 ID',
        speed: '速度',
        exporter: '匯出端',
        local: '匯入為',
        localValue: '匯流排 {{bus}}，位址 {{address}}',
        udc: 'USB 控制器',
        pid: '代理程序編號',
        startedAt: '開始時間',
        isoDevice: '此裝置透過等時端點傳輸資料，這在此硬體上從未測量過',
        exporterLabel: '匯出端位址',
        exporterHint: 'NanoKVM 連線的主機與連接埠。使用下面的通道時即為 {{exporter}}。',
        busIdLabel: '本機上的匯流排 ID',
        busIdHint: 'usbip list -l 為該裝置列出的 busid，例如 {{example}}。',
        start: '啟動直通',
        stop: '停止直通',
        startTitle: '要啟動 USB 直通嗎？',
        startDevice: 'NanoKVM 將從 {{exporter}} 匯入 {{busId}}。',
        startHid: '工作階段執行期間，USB 鍵盤、滑鼠與虛擬媒體會停止運作；停止後會自動恢復。',
        startIso: '網路攝影機等等時傳輸裝置需要在啟動前開啟等時傳輸開關。',
        startWeb: '此網頁介面仍可使用，您隨時可以在本頁停止工作階段。',
        startNetwork:
          '請透過乙太網路或 Wi-Fi 使用本頁。從 NanoKVM 的 USB 網路啟動會被拒絕，因為該連線屆時會消失。',
        okBtn: '啟動',
        cancelBtn: '取消',
        instructions: '在您自己的電腦上',
        instructionsDesc:
          '依照設計，不需要安裝任何用戶端代理。請在擁有該裝置的電腦上執行下列標準 usbip 指令。',
        copyFailed: '複製失敗，請手動複製指令。',
        copyInsecure:
          '本頁面未透過 HTTPS 提供，瀏覽器因此阻擋了複製。請手動複製指令，或在「設定 - 網路」中啟用 HTTPS。',
        directNote:
          '若不使用通道，usbipd 必須在您的網路上可連線，且上面的匯出端位址要指向它。usbip 以明文傳輸裝置資料，建議優先使用通道。',
        steps: {
          modprobe: {
            title: '載入匯出端驅動程式',
            desc: 'usbip-host 讓核心可以把本機裝置交出去，預設不會載入。'
          },
          list: {
            title: '找出裝置',
            desc: '列出所有本機裝置及其 busid 與廠商:產品編號。記下要直通那台裝置的 busid。'
          },
          bind: {
            title: '繫結至 usbip',
            desc: '把裝置從原有驅動程式上取下，在解除繫結之前它在本機上將無法使用。'
          },
          serve: {
            title: '提供服務',
            desc: 'usbipd 會在前景執行，等待 NanoKVM 匯入該裝置。',
            notice:
              '標準 usbipd 沒有監聽位址選項，會監聽所有介面。請在防火牆上關閉連接埠 {{port}}，只允許下面的通道存取。'
          },
          tunnel: {
            title: '指向 NanoKVM',
            desc: 'SSH 反向通道：NanoKVM 本機回送位址上的 {{port}} 連接埠即為本機的 usbipd。整個工作階段期間請保持它執行。'
          },
          exporter: {
            title: '用它作為匯出端',
            desc: '把它填入上面的匯出端位址，輸入匯流排 ID，然後啟動工作階段。'
          },
          unbind: {
            title: '歸還裝置',
            desc: '工作階段停止後，用它把裝置交還給本機原有的驅動程式。'
          }
        }
      },
      mcp: {
        title: 'MCP 服務',
        service: 'MCP 遠端控制',
        serviceDesc: '允許受信任的 MCP 用戶端控制鍵盤、滑鼠並擷取螢幕截圖',
        securityWarning:
          '任何持有此 API Key 的人都可以控制遠端主機並查看螢幕。請使用 HTTPS，且只在受信任的網路中啟用。',
        endpoint: '服務位址',
        apiKey: 'API Key',
        regenerateConfirmTitle: '重新產生 MCP API Key？',
        regenerateConfirmDesc: '目前的 Key 將立即失效。',
        enableConfirmTitle: '啟用外部 MCP 控制？',
        enableConfirmDesc: '啟用 MCP 將停止 PicoClaw，並關閉目前作用中的 PicoClaw 工作階段。',
        failed: 'MCP 操作失敗',
        copyFailed: '複製失敗，請手動複製。',
        copyInsecure:
          '本頁面未透過 HTTPS 提供，瀏覽器因此阻擋了複製。請手動複製，或在「設定 - 網路」中啟用 HTTPS。',
        okBtn: '確認',
        cancelBtn: '取消'
      },
      about: {
        title: '關於 NanoKVM',
        information: '資訊',
        ip: 'IP',
        mdns: 'mDNS',
        application: '應用程式版本',
        applicationTip: 'NanoKVM 網頁程式版本',
        image: '韌體版本',
        imageTip: 'NanoKVM 系统韌體版本',
        deviceKey: '設備序號',
        community: '社群',
        hostname: '主機名稱',
        hostnameUpdated: '已更新主機名稱. 請重新啟動以生效',
        ipType: {
          Wired: '有線',
          Wireless: '無線',
          Other: '其他'
        }
      },
      appearance: {
        title: '外觀',
        display: '顯示',
        language: '語言',
        languageDesc: '選擇介面語言',
        webTitle: '網頁標題',
        webTitleDesc: '自訂網頁標題',
        favicon: '網頁圖示',
        faviconDesc: '自訂瀏覽器分頁圖示',
        faviconPlaceholder: '圖片網址',
        faviconUpload: '上傳',
        faviconReset: '重設',
        faviconCustom: '自訂圖示',
        faviconBoot: '來自 /boot/logo.ico 的圖示',
        faviconDefault: '預設圖示',
        faviconOverridesBoot: '已覆寫 /boot/logo.ico',
        faviconErrUrl: '請輸入 http:// 或 https:// 開頭的圖片網址',
        faviconErrFetch: '裝置無法下載該圖片',
        faviconErrLarge: '圖片過大，上限為 256 KB',
        faviconErrType: '不支援的圖片格式，請使用 .ico、.png、.svg、.gif 或 .jpg',
        faviconErrSave: '儲存圖示失敗',
        menuBar: {
          title: '選單列',
          mode: '顯示方式',
          modeDesc: '選單欄在螢幕上的顯示方式',
          modeOff: '關閉',
          modeAuto: '自動隱藏',
          modeAlways: '始終顯示',
          keyboardLedStatus: '鍵盤鎖定狀態指示燈',
          keyboardLedStatusDesc: '顯示遠端電腦的 Num Lock、Caps Lock 與 Scroll Lock 狀態',
          icons: '選單圖示',
          iconsDesc: '是否在選單欄中顯示子選單圖示'
        }
      },
      keyboardLedStatus: {
        groupLabel: '遠端鍵盤鎖定狀態',
        indicatorLabel: '{{label}}：{{state}}',
        numLock: '數字鎖定',
        numLockShort: '數',
        capsLock: '大寫鎖定',
        capsLockShort: '大',
        scrollLock: '捲動鎖定',
        scrollLockShort: '捲',
        on: '開啟',
        off: '關閉',
        unknown: '未知'
      },
      device: {
        title: '設備',
        oled: {
          title: 'OLED',
          description: '設定 OLED 螢幕自動睡眠時間',
          0: '永不',
          15: '15 秒',
          30: '30 秒',
          60: '1 分鐘',
          180: '3 分鐘',
          300: '5 分鐘',
          600: '10 分鐘',
          1800: '30 分鐘',
          3600: '1 小時'
        },
        ssh: {
          description: '啟用 SSH 伺服器',
          tip: '啟用前請務必設定強密碼（帳號 - 更改密碼）'
        },
        advanced: '進階設定',
        swap: {
          title: 'Swap',
          disable: '停用',
          description: '設定 Swap 檔大小',
          tip: '啟用此功能可能會減少SD卡的使用壽命！'
        },
        mouseJiggler: {
          title: '滑鼠抖動模式 (Mouse Jiggler)',
          description: '避免遠端主機進入休眠狀態',
          disable: '停用',
          absolute: '絕對模式',
          relative: '相對模式'
        },
        mdns: {
          description: '啟用 mDNS 發現服務',
          tip: '若無需求，建議關閉此功能'
        },
        hdmi: {
          description: '啟用 HDMI/螢幕 輸出',
          idleTimeoutTitle: '擷取閒置逾時',
          idleTimeoutDescription: '沒有活躍觀看者時，在指定時間後停止 HDMI 擷取',
          minutes: '分鐘'
        },
        autostart: {
          title: '啟動時指令碼設定',
          description: '管理能夠在 NanoKVM 啟動時自動執行的相關指令碼',
          new: '建立新指令碼',
          deleteConfirm: '確定要刪除該檔案嗎？',
          yes: '是',
          no: '否',
          scriptName: 'Script 名稱',
          scriptContent: 'Script 內容',
          settings: '設定'
        },
        hidOnly: 'HID-Only 模式',
        hidOnlyDesc: '停止模擬虛擬設備，僅保留基礎 HID 控制',
        disk: '虛擬隨身碟',
        diskDesc: '在遠端主機上連接虛擬隨身碟',
        rebindNotice: '切換此開關會讓 USB 裝置重新列舉，目標主機會短暫失去虛擬裝置與 USB 網路。',
        media: {
          title: '攝影機、麥克風與喇叭插槽',
          desc: '宣告目標主機看到的媒體裝置。端點預算會在套用 USB 設定檔時檢查。 儲存會重新列舉裝置，並中斷已連線的瀏覽器。',
          cameras: '攝影機',
          microphones: '麥克風',
          speakers: '喇叭',
          name: '名稱',
          namePlaceholder: '顯示在目標主機上',
          addCamera: '新增攝影機',
          addMicrophone: '新增麥克風',
          addSpeaker: '新增喇叭',
          remove: '移除',
          cameraDefault: '攝影機 {{index}}',
          microphoneDefault: '麥克風 {{index}}',
          speakerDefault: '喇叭 {{index}}',
          nameRequired: '每個插槽都需要名稱。',
          budgetHint:
            '六個 USB IN 端點是固定的硬體上限。請在「USB 呈現」中把鍵盤、滑鼠與絕對定位放到同一個 HID 介面，或在此關閉虛擬磁碟，或在「網路」中關閉 USB 網卡。',
          unsupported: '此核心無法為媒體裝置命名，因此主機會顯示預設名稱。',
          save: '儲存插槽',
          disconnect: '中斷連線',
          disconnectAll: '中斷所有來源',
          limit: '攝影機、麥克風與喇叭插槽合計不得超過八個。',
          failed: '無法更新媒體插槽。'
        },
        reboot: '重新啟動',
        rebootDesc: '您確定要重新啟動 NanoKVM?',
        okBtn: '確定',
        cancelBtn: '取消',
        diskLocked: '掛載映像檔期間已關閉（開啟會取消映像檔掛載）',
        diskCancelMount:
          '開啟虛擬隨身碟會取消目前掛載的映像檔，並中斷正在進行的映像檔下載或上傳，是否繼續？',
        diskBreakDownload: '開啟虛擬隨身碟會中斷映像檔下載功能，是否繼續？',
        attention: '注意',
        updateFailed: '虛擬裝置更新失敗'
      },
      network: {
        title: '網路',
        wifi: {
          title: 'Wi-Fi',
          description: '設定 Wi-Fi',
          apMode: 'AP 模式已啟用，請掃描 QRCode 連接 Wi-Fi',
          connect: '連線 Wi-Fi',
          connectDesc1: '請輸入網路名稱和密碼',
          connectDesc2: '請輸入密碼以連線此網路',
          disconnect: '是否要中斷該網路連線？',
          failed: '連線失敗，請重試',
          ssid: 'SSID 名稱',
          password: '密碼',
          joinBtn: '加入',
          confirmBtn: '確定',
          cancelBtn: '取消'
        },
        tls: {
          description: '啟用 HTTPS 協議',
          tip: '啟用 HTTPS 可以提高安全性，但可能會增加傳輸延遲，特別是使用 MJPEG 格式傳輸時。'
        },
        usb: {
          title: 'USB 網卡',
          desc: '透過 USB 給被控電腦一張網卡',
          type: '網卡類型',
          typeDesc: '新式系統使用 NCM，較舊的 Windows 使用 RNDIS'
        },
        bridge: {
          title: '網卡連接到',
          lan: '你的網路',
          kvmOnly: '僅 NanoKVM',
          lanDesc: '被控電腦經 NanoKVM 的網路埠加入你的網路，並從路由器取得自己的位址。',
          kvmOnlyDesc: '被控電腦從 NanoKVM 取得位址，能連到 NanoKVM，但到不了更遠的地方。',
          loading: '載入中...',
          state: '狀態',
          states: {
            disabled: '僅 NanoKVM',
            enabled: '你的網路',
            rolledBack: '已還原',
            failed: '失敗',
            pending: '進行中'
          },
          uplink: '上行介面',
          ports: '連接埠',
          up: '已連線',
          down: '未連線',
          noLink: '無連線',
          enableTitle: '將被控電腦連上你的網路？',
          disableTitle: '將被控電腦限制在僅 NanoKVM？',
          reconnect: '位址移轉期間，管理連線會短暫中斷並重新連線。',
          rollback: '若驗證失敗，將自動還原先前的網路設定。',
          enableBtn: '連上我的網路',
          disableBtn: '僅 NanoKVM',
          cancelBtn: '取消',
          interrupted: '套用過程中連線中斷，正在重新檢查目前狀態。',
          pendingNotice: '橋接變更仍在進行中，或在完成前被中斷。',
          revert: '還原先前的設定',
          rolledBackNotice: '上次變更已還原，先前的網路設定已恢復。',
          verifyFailed: '驗證失敗：{{gates}}',
          gates: {
            address: '位址',
            gateway: '閘道',
            inbound: '傳入連線'
          },
          inboundWeak:
            '入站檢查僅透過 NanoKVM 自我連線完成。這只能證明 Web 服務正在接聽且本機可達，並不能證明來自網路的請求可以抵達。',
          noCarrier: '{{port}} 上沒有連線。在接上網路線之前，橋接沒有通往網路的路徑。',
          loop: '路由器同時也在 {{port}} 上被學習到，代表該連接埠是通往同一網路的第二條路徑。生成樹已關閉，這裡不會有任何機制打斷迴圈：請拔除其中一條路徑。',
          failedNotice: '上次變更無法復原。可能只能透過 Wi-Fi AP 或序列主控台存取 NanoKVM。'
        },
        dns: {
          title: 'DNS',
          description: '設定 NanoKVM 使用的 DNS 伺服器',
          mode: '模式',
          dhcp: 'DHCP',
          manual: '手動',
          add: '新增 DNS',
          save: '儲存',
          invalid: '請輸入有效的 IP 位址',
          noDhcp: '目前未取得 DHCP DNS',
          saved: 'DNS 設定已儲存',
          saveFailed: '儲存 DNS 設定失敗',
          unsaved: '有未儲存的變更',
          maxServers: '最多允許 {{count}} 個 DNS 伺服器',
          dnsServers: 'DNS 伺服器',
          dhcpServersDescription: 'DNS 伺服器由 DHCP 自動取得',
          manualServersDescription: 'DNS 伺服器可以手動編輯',
          networkDetails: '網路詳細資訊',
          interface: '介面',
          ipAddress: 'IP 位址',
          subnetMask: '子網路遮罩',
          router: '路由器',
          none: '無'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC 伺服器',
        description:
          '使用您的 NanoKVM 帳號登入後，任何 VNC 用戶端都可以檢視遠端畫面並使用鍵盤與滑鼠',
        port: '連接埠',
        portDescription: '連線至 NanoKVM 位址上的這個連接埠'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: '記憶體最佳化',
          tip: '當記憶體使用量超過限制時，會更積極的進行垃圾回收來嘗試釋放記憶體。若使用 Tailscale 建議設定為 50MB，於重啟 Tailscale 後生效。'
        },
        swap: {
          title: 'Swap',
          tip: '如果啟用記憶體最佳化後依然存在問題，可以嘗試開啟 Swap。啟用後會將交換檔案設定為 256MB，可以在「設定 - 裝置」中修改該選項。'
        },
        restart: '確定要重啟 Tailscale 嗎？',
        stop: '確定要停止 Tailscale 嗎？',
        stopDesc: '登出 Tailscale 並停用開機自動啟動。',
        loading: '載入中...',
        notInstall: '未找到 Tailscale ！請先安裝。',
        install: '安裝',
        installing: '安裝中',
        failed: '安裝失敗',
        retry: '請重新整理並重試。或嘗試手動安裝',
        download: '下載',
        package: '安裝包',
        unzip: '並解壓縮它',
        upTailscale: '將 Tailscale 上傳到 NanoKVM 的 /usr/bin/ 資料夾',
        upTailscaled: '將 Tailscale 上傳到 NanoKVM 的 /usr/sbin/ 資料夾',
        refresh: '重新整理頁面',
        notRunning: 'Tailscale 尚未執行',
        run: '啟動',
        notLogin: '設備尚未綁定。請登入並將該裝置綁定到您的帳戶。',
        urlPeriod: '此網址有效期限為 10 分鐘',
        login: '登入',
        loginSuccess: '登入成功',
        enable: '啟用 Tailscale',
        deviceName: '裝置名稱',
        deviceIP: '裝置 IP',
        account: '帳號',
        logout: '登出',
        logoutDesc: '確認要登出嗎？',
        uninstall: '移除 Tailscale',
        uninstallDesc: '確定要解除安裝 Tailscale 嗎？',
        okBtn: '確認',
        cancelBtn: '取消'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: '載入中...',
        notInstall: '尚未安裝',
        notConfigured: '尚未設定',
        stopped: '已停止',
        running: '執行中',
        connected: '已連線',
        error: '錯誤',
        atBoot: '開機時啟動',
        notAtBoot: '開機時不啟動',
        arguments: '啟動參數',
        argumentsTip: '啟動服務時傳入的命令列參數。',
        env: '環境變數',
        envKey: '名稱',
        envValue: '值',
        envAdd: '新增變數',
        envRemove: '移除',
        configured: '已設定',
        save: '儲存',
        saved: '設定已儲存',
        start: '啟動',
        stop: '停止',
        restart: '重新啟動',
        logs: '日誌',
        logsEmpty: '尚無日誌',
        refresh: '重新整理',
        binary: '執行檔',
        binaryShipped: '韌體內建',
        binaryCustom: '自行上傳',
        binaryUpload: '上傳執行檔',
        binaryRevert: '還原內建執行檔',
        binaryRevertDesc: '確定要刪除已上傳的執行檔並還原韌體內建的版本嗎？',
        serverWarning: '未加限制的伺服器等同開放式代理',
        noHealthSignal:
          '此服務不提供健康狀態訊號，因此 NanoKVM 只能確認行程正在執行，無法確認通道是否已連線。',
        memoryWarning: '同時執行多個遠端存取服務可能會耗盡記憶體',
        resources: '資源',
        memory: {
          title: '記憶體上限',
          description:
            '自下次重新啟動起將 newt 的 Go 堆積限制為 {{limit}} MiB。這是它自己的上限，而非 Tailscale 的；關閉時保留 Go 預設值，兩種情況都會套用 GOGC=50。',
          noRuntime:
            'wstunnel 以 Rust 撰寫：沒有垃圾回收器，也沒有可設定的堆積上限，其工作執行緒本來就跟隨裝置的單一 CPU。',
          notApplicable: '不適用'
        },
        swap: {
          title: '交換檔',
          description:
            '在 SD 卡上新增 256 MB 的交換檔。作用於整個系統：同一份交換空間供 Tailscale、KVM 伺服器以及裝置上的其他一切使用。'
        },
        okBtn: '確定',
        cancelBtn: '取消'
      },
      exit: {
        title: '出口隧道',
        description:
          '把 USB 網路介面變成目標機的上網出口：目標機的流量經由透過此網頁位址接入的你的裝置流出。',
        loading: '載入中...',
        noSlots: '此裝置未設定任何出口插槽。',
        slot: '插槽 {{slot}}',
        enable: '出口隧道',
        pending: '正在套用...',
        statusStale: '狀態未更新',
        enableConfirm: '啟用出口隧道？',
        disableConfirm: '停用出口隧道？',
        reenumerate:
          '僅在出口裝置連線期間，目標機才會取得本 NanoKVM 作為閘道和 DNS。出口裝置連線時，或斷線滿 15 秒時，USB 裝置將重新列舉，讓目標機取得或移除它們。鍵盤、滑鼠、攝影機和虛擬磁碟會中斷約兩秒。',
        okBtn: '確定',
        cancelBtn: '取消',
        copy: '複製',
        copied: '已複製',
        copyFailed: '複製失敗，請選取文字手動複製',
        state: {
          disconnected: '未連線',
          connecting: '連線中',
          connected: '已連線'
        },
        status: {
          title: '狀態',
          disconnect: '中斷出口',
          disconnectDesc: '中斷已連線的出口裝置。除非先停止其用戶端，否則它會自動重新連線。',
          tunnel: '隧道',
          uptime: '已執行 {{uptime}}',
          lastConnected: '上次連線 {{time}}',
          never: '從未',
          peer: '出口裝置',
          noPeer: '沒有已連線的出口裝置',
          peerChanged: '於 {{time}} 從 {{addr}} 切換',
          nic: '目標機網卡',
          nicUp: '已啟用',
          nicDown: '未啟用',
          nicMissing: '尚未識別',
          nicNone: '目前的 USB 設定檔不含網路功能。請在「網路」中開啟 USB 網卡，為目標機提供網卡。',
          upstream: '網際網路',
          reachable: '可達',
          unreachable: '不可達',
          latency: '{{ms}} ms',
          notProbed: '出口連線前不探測',
          downstream: '下行鏈路',
          downstreamOff: '隧道停用期間關閉',
          downstreamDegraded:
            '下行鏈路缺少了一部分。看門狗每 30 秒重試一次；下方日誌會說明失敗原因。',
          downstreamTip: {
            forward: 'IPv4 轉送已開啟',
            routing: '策略路由規則與圍欄路由已就位',
            tun: 'tun 裝置存在且有讀取端接入',
            hev: 'tun2socks 轉換器正在執行',
          wstunnel: 'wstunnel 伺服器正在執行（原生模式下始終為真）',
            dns: 'DNS 轉送器已繫結到 USB 網路位址',
            nat: '防火牆鏈已存在並已掛接'
          },
          dns: 'DNS',
          dnsStats: '{{queries}} 次查詢，{{failures}} 次失敗，{{redirected}} 次重新導向',
          traffic: '流量',
          trafficUp: '送往出口 {{bytes}}',
          trafficDown: '來自出口 {{bytes}}'
        },
        token: {
          title: '權杖',
          history: '包含在下方每條命令中，因此會留在出口裝置的 shell 歷史紀錄裡。',
          regenerate: '重新產生',
          regenerateConfirm: '重新產生權杖？',
          regenerateDesc: '請先停止正在執行的用戶端。所有已連線的出口都會被中斷，舊命令將失效。'
        },
        mode: {
          title: '模式',
          native: '原生',
          wstunnel: 'wstunnel',
          nexit: 'nexit',
          nativeDesc:
            '建議。從此 NanoKVM 取得的小型指令碼在出口裝置上執行，只用作業系統內建的工具（PowerShell、Python 或 Perl），以一般使用者身分執行，無需安裝，也不需要系統管理員權限。',
          nexitDesc:
            'Windows. The same client as a single binary, for a machine with no usable PowerShell. It runs from the Command Prompt with nothing else involved, and uses the same slot as the script.',
          wstunnelDesc:
            '替代方案。出口裝置下載固定版本 {{version}} 的二進位檔，並在執行前驗證其 SHA-256。',
          whileEnabled: '隧道啟用時切換模式會中斷已連線的出口裝置，請以新模式的命令重新啟動。'
        },
        commands: {
          nexitFiles: {
            title: 'Or download nexit and its config file',
            x64: 'Download nexit (x64)',
            arm64: 'Download nexit (ARM64)',
            config: 'Download nexit.json',
            instructions:
              'Put nexit.json next to nexit.exe, or in %APPDATA%\\nexit\\, or (as administrator) in %ProgramData%\\nexit\\, then double-click nexit.exe.',
            secret:
              "nexit.json contains this slot's passcode, like the commands above. Keep it private; regenerating the token makes it useless.",
            needsEnabled: 'Enable the slot to download these files.',
            failed: 'Download failed: {{error}}'
          },
          title: '在出口裝置上執行',
          description:
            '在你希望目標機借用其網路的那台機器上貼上一條命令。它會持續重新連線，直到你停止它。',
          warnDownload: '該命令會從此 NanoKVM 下載指令碼，並以你的使用者身分執行。',
          warnSecret: '它包含控制目標機上網的密鑰，並會儲存在你的 shell 歷史紀錄中。',
          warnReach: '出口裝置能存取的一切，目標機也將能存取。',
          warnCleartext: '本頁透過明文 http 提供：權杖以及出口與 NanoKVM 之間的所有流量均未加密。',
          warnTransportTrusted:
            "This NanoKVM's certificate is self-signed, so the command trusts the connection rather than verifying it and the token is what authenticates the session. Install a CA-signed certificate and the commands verify it instead.",
          warnWstunnelDefender:
            'Windows Defender 可能會隔離下載的 wstunnel 二進位檔並中斷隧道，且不在其記錄中留下任何內容；原生命令不會下載二進位檔。',
          shell: {
            powershell: 'PowerShell',
            cmd: 'Command Prompt',
            bash: 'Bash'
          },
          windows: 'Windows',
          macos: 'macOS',
          linux: 'Linux',
          rewritten: '已改為指向 {{host}}，即本瀏覽器存取的位址。',
          warnSchemeMismatch:
            'NanoKVM 看到此頁面是透過 {{server}} 請求的，但瀏覽器是經由 {{local}} 存取。命令指向 {{server}}；請讓代理伺服器傳送 X-Forwarded-Proto，命令便會採用您使用的位址。',
          unavailable: '該平台沒有可用命令。',
          regenerateHint:
            '已有出口連線。不再需要貼上的命令後，請重新產生權杖；之後需用新命令重新啟動出口。',
          security:
            '目標機本來就能透過 USB 網路位址存取此 NanoKVM 的網頁介面、SSH 和 VNC。此功能不改變這一點。',
          latestTitle: '最新 wstunnel',
          latestDesc:
            '命令執行時解析 GitHub 上最新的 wstunnel 發行版本，並以該版本的 checksums.txt 驗證下載檔案，因此它不是固定版本 {{version}}，也不帶固定雜湊。',
          latestFallback: '如果取得失敗，請從 github.com/erebe/wstunnel 下載'
        },
        advanced: {
          title: '進階',
          dns: 'DNS 伺服器',
          dnsTip:
            '最多四個解析器，經由出口存取。目標機會把 NanoKVM 當作解析器，所有明文 DNS 查詢都會被重新導向到這裡。',
          dnsAdd: '新增伺服器',
          dnsRemove: '移除伺服器',
          dnsInvalid: '只能輸入 IP 位址',
          dnsDenied: '無法透過出口存取 loopback、link-local、多播及 198.18/15 位址',
          dnsPrivate: '使用私有解析器需要開啟「允許私有目的地」',
          mtu: 'MTU',
          mtuTip: '{{min}} 到 {{max}}。1280 可為隧道標頭留出空間。',
          allowPrivate: '允許私有目的位址',
          allowPrivateTip:
            '允許目標機存取出口裝置自己的區域網路：10/8、172.16/12、192.168/16、100.64/10 和 fc00::/7。關閉時這些位址會被拒絕。',
          pinPeer: '固定到首個出口',
          pinPeerTip: '在關閉前拒絕來自其他位址的連線。否則新的有效連線會取代舊連線。',
          save: '儲存',
          saved: '已儲存'
        },
        logs: {
          title: '日誌',
          empty: '尚無日誌',
          refresh: '重新整理',
          hev: 'tun2socks (hev-socks5-tunnel)',
          wstunnel: 'wstunnel 伺服器'
        }
      },
      update: {
        title: '檢查更新',
        queryFailed: '取得版本號失敗',
        updateFailed: '更新失敗。請重試。',
        isLatest: '您已經是最新版本。',
        rebooting: '正在安裝新核心並重新開機，可能需要幾分鐘，請勿斷電。',
        kernelUpdate:
          '此更新將安裝核心 {{version}}。裝置會重新開機；若新核心無法啟動，會自動回復到目前的核心。',
        rolledBack: '核心 {{version}} 啟動失敗，裝置已回復到上一個核心。',
        available: '發現有可用更新。您確定要更新嗎？',
        updating: '更新中，請稍候...',
        confirm: '確定',
        cancel: '取消',
        preview: '預覽更新',
        previewDesc: '預覽版本，搶先體驗新功能和改進',
        previewTip: '請注意，預覽版本可能包含一些不穩定因素或未完善的功能！',
        customServer: {
          title: '自訂更新伺服器',
          desc: '從指定伺服器檢查並下載線上更新',
          invalidUrl:
            '請輸入有效的 HTTP 或 HTTPS 伺服器目錄，不可包含查詢參數、片段或 latest.json。',
          loadFailed: '讀取更新伺服器設定失敗。',
          saveFailed: '儲存更新伺服器設定失敗。',
          saved: '更新伺服器設定已儲存。',
          save: '儲存',
          confirmTitle: '使用自訂更新伺服器？',
          confirmDesc:
            'SHA-512 只能驗證安裝套件與該伺服器提供的清單一致，不能證明安裝套件來自 NanoKVM 官方。錯誤或惡意的伺服器可能導致裝置無法使用、資料遺失或系統遭到接管。',
          confirm: '仍然使用',
          previewDisabled: '啟用自訂更新伺服器時，預覽更新無法使用'
        },
        offline: {
          kernelNotice:
            '此安裝包含核心。核心會寫入備用插槽並重新開機試執行；若無法啟動，裝置會自動回復到目前的核心。',
          kernelConfirm: '安裝核心',
          kernelCancel: '取消',
          title: '離線更新',
          desc: '透過本地安裝包進行更新',
          upload: '上傳',
          checksumPlaceholder: 'SHA-256 校驗和（選填）',
          invalidChecksum: 'SHA-256 校驗和必須包含 64 個十六進位字元。',
          checksumMismatch: 'SHA-256 驗證失敗。套件可能已損毀。',
          invalidName: '檔名格式錯誤，請前往 GitHub 釋出頁下載安裝包。',
          updateFailed: '更新失敗，請重試'
        }
      },
      account: {
        title: '帳號',
        webAccount: '網頁帳號',
        password: '密碼',
        updateBtn: '修改',
        logoutBtn: '登出',
        logoutDesc: '您確定要登出嗎?',
        okBtn: '確定',
        cancelBtn: '取消',
        role: '角色',
        roles: {
          admin: '管理員',
          user: '一般使用者'
        },
        users: {
          title: '使用者管理',
          create: '建立使用者',
          enabled: '已啟用',
          disabled: '已停用',
          deviceOwner: '裝置擁有者',
          resetPassword: '重設密碼',
          delete: '刪除',
          deleteConfirm: '刪除此使用者並撤銷其全部工作階段？',
          created: '使用者已建立',
          deleted: '使用者已刪除',
          passwordUpdated: '密碼已更新',
          loadFailed: '載入使用者失敗',
          saveFailed: '儲存使用者失敗',
          deleteFailed: '刪除使用者失敗',
          editPermissions: '權限',
          mcpLossTitle: '撤銷 MCP 存取',
          mcpLossDesc:
            '{{name}} 將失去 MCP 存取權限。MCP API Key 由整台裝置共用，已複製出去的 Key 在重新產生前仍然有效。重新產生會中斷所有 MCP 用戶端，直到它們改用新 Key。',
          mcpLossRotate: '重新產生 MCP API Key',
          mcpLossKeep:
            '目前的 Key 將繼續有效，持有副本的人仍可控制鍵盤和滑鼠。之後可在 MCP 設定中重新產生。',
          mcpKeyRotated: 'MCP API Key 已重新產生，MCP 用戶端需改用新 Key。'
        },
        permissions: {
          title: '功能權限',
          editTitle: '編輯權限',
          save: '儲存',
          saved: '已儲存',
          close: '關閉',
          groupConsole: '控制台',
          groupTarget: '被控機',
          groupLibrary: '媒體庫',
          groupHost: '本裝置',
          groupServices: '服務',
          consoleView: '觀看畫面',
          consoleControl: '鍵鼠操作',
          power: '電源與喚醒',
          mediumAttach: '掛載與交換媒體',
          libraryRead: '瀏覽媒體庫',
          libraryWrite: '管理媒體庫',
          terminal: '終端機與腳本',
          device: '裝置設定',
          network: '網路設定',
          update: '系統更新',
          mcp: 'MCP 服務',
          tailscale: 'Tailscale',
          ai: 'AI 助手',
          effectConsoleView:
            '顯示被控機畫面，並設定所有觀看者共用的視訊模式、解析度、影格率與畫質。',
          effectConsoleControl: '向被控機發送鍵盤與滑鼠事件。',
          effectPower: '對被控機開機、關機、重新啟動，並管理網路喚醒裝置清單。',
          effectMediumAttach: '把媒體掛載到被控機或退出，並透過共享分割區與被控機交換檔案。',
          effectLibraryRead: '列出並下載映像檔及映像檔內的檔案。',
          effectLibraryWrite: '上傳、建立、刪除映像檔，並修改映像檔內的檔案。',
          effectTerminal: '在本裝置上取得 root shell，無需經過 Web 工作階段。',
          effectDevice:
            '可啟用 SSH（同樣是 root）、重新啟動裝置、把原始資料分割區交給被控機，並修改 OLED、HDMI、mDNS、記憶體與置換空間設定。',
          effectNetwork: '修改 Wi-Fi、DNS 與 TLS，可能使本裝置離線。',
          effectUpdate: '安裝韌體與應用程式更新包。',
          effectMcp:
            '下發裝置級 API Key，可在 Web 工作階段之外控制鍵鼠並取得截圖；撤銷此權限後，已複製的 Key 仍有效，除非重新產生（移除權限時會詢問是否重新產生）。',
          effectTailscale: '把本裝置加入 VPN 網路。',
          effectAi: '透過不受其他權限限制的本機介面操作裝置，因此相當於完整的裝置控制權。',
          fullControlWarning: '下列權限等同於完整的裝置控制權：',
          emptyWarning: '此帳戶將不具備任何功能，只能登入。',
          impliedBy: '由「{{name}}」自動包含',
          grantCount_one: '{{count}} 項權限',
          grantCount: '{{count}} 項權限',
          adminAll: '管理員擁有全部權限',
          catalogFailed: '權限清單載入失敗，請關閉後重試。'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw 助理',
      empty: '打開面板並啟動一個任務來開始。',
      inputPlaceholder: '描述您希望 PicoClaw 執行的操作',
      newConversation: '新對話',
      processing: '正在處理...',
      agent: {
        defaultTitle: '通用助理',
        defaultDescription: '一般聊天、搜尋和工作區域幫助。',
        kvmTitle: '遠端控制',
        kvmDescription: '透過 NanoKVM 操作遠端主機。',
        switched: '代理角色已切換',
        switchFailed: '代理角色切換失敗'
      },
      send: '發送',
      cancel: '取消',
      status: {
        connecting: '正在連線 Gateway...',
        connected: 'PicoClaw 會話已連線',
        disconnected: 'PicoClaw 會話已關閉',
        stopped: '已發送停止請求',
        runtimeStarted: 'PicoClaw Runtime 已啟動',
        runtimeStartFailed: '啟動 PicoClaw Runtime 失敗',
        runtimeStopped: 'PicoClaw Runtime 已停止',
        runtimeStopFailed: '停止 PicoClaw Runtime 失敗',
        controlSwitchedToMCP: '控制權已切換至外部 MCP 服務'
      },
      connection: {
        runtime: {
          checking: '檢查中',
          restoring: '正在恢復 PicoClaw',
          ready: 'Runtime 已就緒',
          stopped: 'Runtime 未啟動',
          blockedByMCP: '外部 MCP 控制已啟用',
          readyBlockedByMCP: 'Runtime 正在執行，但外部 MCP 目前控制裝置輸入。',
          readyWithoutControl: 'Runtime 正在執行，請先授予 PicoClaw 裝置控制權後再重新連線。',
          unavailable: 'Runtime 不可用',
          configError: '設定錯誤'
        },
        transport: {
          connecting: '連接中',
          connected: '已連接',
          disconnected: '未連線',
          reconnect: '重新連線',
          reconnectDescription: '重新連線到正在執行的 PicoClaw 會話。',
          reconnectBlocked: 'PicoClaw 需要先取得裝置控制權才能重新連線。'
        },
        run: {
          idle: '空閒',
          busy: '忙'
        }
      },
      message: {
        toolAction: '行動',
        observation: '觀察',
        screenshot: '截圖'
      },
      overlay: {
        locked: 'PicoClaw 正在控制設備。手動輸入已暫停。'
      },
      control: {
        picoclaw: '設備控制：PicoClaw',
        picoclawDescription: 'PicoClaw 可以寫入鍵鼠，手動輸入可能會被暫停。',
        mcp: '設備控制：外部 MCP',
        mcpDescription: '外部 MCP 可以寫入裝置，PicoClaw 不會接管鍵鼠。',
        off: '設備控制：關閉',
        offDescription: 'AI 不會寫入鍵鼠，手動控制保持可用。',
        transitioning: '裝置控制：正在切換',
        transitioningDescription: '正在同步裝置控制權，請稍候。',
        grant: '授予控制權',
        release: '釋放',
        releasing: '正在釋放...',
        switching: '正在切換...',
        releasingLabel: '裝置控制：正在釋放',
        releasingDescription: '正在交還裝置控制，PicoClaw 已停止目前寫入。',
        granted: '已授予 PicoClaw 控制權',
        released: '已釋放 PicoClaw 控制權',
        grantFailed: '授予 PicoClaw 控制權失敗',
        releaseFailed: '釋放 PicoClaw 控制權失敗',
        grantConfirmTitle: '將設備控制切換至 PicoClaw？',
        grantConfirmDesc: '外部 MCP 的設備寫入將被中斷。'
      },
      install: {
        install: '安裝 PicoClaw',
        installing: '正在安裝 PicoClaw',
        success: 'PicoClaw 安裝成功',
        failed: 'PicoClaw 安裝失敗',
        uninstalling: '正在解除安裝 Runtime...',
        uninstalled: 'Runtime 解除安裝成功。',
        uninstallFailed: '解除安裝失敗。',
        requiredTitle: 'PicoClaw 未安裝',
        requiredDescription: '在啟動 PicoClaw Runtime 之前，請先安裝 PicoClaw。',
        progressDescription: '正在下載並安裝 PicoClaw。',
        stages: {
          preparing: '準備中',
          downloading: '下載中',
          extracting: '解壓縮中',
          verifying: '驗證中',
          installing: '安裝中',
          installed: '已安裝',
          install_timeout: '超時',
          install_failed: '失敗'
        }
      },
      model: {
        requiredTitle: '需要設定模型',
        requiredDescription: '在使用 PicoClaw 聊天之前，請先設定 PicoClaw 模型。',
        docsTitle: '設定指南',
        docsDesc: '支援的模型與通訊協定',
        menuLabel: '設定模型',
        modelIdentifier: '模型標識符',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API Key',
        apiKeyPlaceholder: '請輸入模型 API Key',
        save: '儲存',
        saving: '儲存中',
        saved: '模型設定已儲存',
        saveFailed: '儲存模型設定失敗',
        invalid: '模型標識、API Base URL 和 API Key 不能為空'
      },
      uninstall: {
        menuLabel: '解除安裝',
        confirmTitle: '解除安裝 PicoClaw',
        confirmContent: '您確定要解除安裝 PicoClaw 嗎？這將刪除可執行檔和所有設定檔。',
        confirmOk: '解除安裝',
        confirmCancel: '取消'
      },
      history: {
        title: '歷史會話',
        loading: '正在載入會話...',
        emptyTitle: '還沒有歷史記錄',
        emptyDescription: '之前的 PicoClaw 會話將會出現在此。',
        loadFailed: '無法載入會話歷史記錄',
        deleteFailed: '刪除會話失敗',
        deleteConfirmTitle: '刪除會話',
        deleteConfirmContent: '您確定要刪除「{{title}}」嗎？',
        deleteConfirmOk: '刪除',
        deleteConfirmCancel: '取消',
        messageCount_one: '{{count}} 則訊息',
        messageCount_other: '{{count}} 則訊息',
        messageCount: '{{count}} 則訊息'
      },
      config: {
        startRuntime: '啟動 PicoClaw',
        stopRuntime: '停止 PicoClaw'
      },
      start: {
        enableConfirmTitle: '將控制權切換至 PicoClaw？',
        enableConfirmDesc: '啟動 PicoClaw 將停用外部 MCP 服務。',
        enableConfirmOk: '啟動 PicoClaw',
        enableConfirmCancel: '取消',
        title: '啟動 PicoClaw',
        description: '啟動 Runtime 後即可開始使用 PicoClaw 助理。',
        switchFromMCP: '切換到 PicoClaw 並啟動',
        takeoverAndStart: '接管並啟動'
      }
    },
    error: {
      title: '我們遇到了一些問題',
      refresh: '重新整理',
      panel: '此面板已停止運作',
      retry: '重試',
      reload: '重新載入頁面'
    },
    fullscreen: {
      toggle: '進入全螢幕模式'
    },
    menu: {
      collapse: '收起選單',
      expand: '展開選單'
    }
  }
};

export default zh_tw;
