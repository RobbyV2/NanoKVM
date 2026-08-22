const zh = {
  translation: {
    head: {
      desktop: '远程桌面',
      login: '登录',
      changePassword: '修改密码',
      terminal: '终端',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: '登录',
      placeholderUsername: '请输入用户名',
      placeholderPassword: '请输入密码',
      placeholderCurrentPassword: '请输入当前密码',
      placeholderPassword2: '请再次输入密码',
      noEmptyUsername: '用户名不能为空',
      noEmptyPassword: '密码不能为空',
      passwordLength: '密码长度必须为 8 到 72 个字符',
      noAccount: '获取用户信息失败，请刷新重试或重置密码',
      invalidUser: '用户名或密码错误',
      locked: '登录太频繁，请稍后再试',
      globalLocked: '系统防爆破保护中，请稍后再试',
      error: '未知错误',
      invalidCurrentPassword: '当前密码错误',
      changePassword: '修改密码',
      changePasswordDesc: '为了您的设备安全，请修改密码!',
      differentPassword: '两次密码不一致',
      illegalUsername: '用户名中包含非法字符',
      illegalPassword: '密码中包含非法字符',
      forgetPassword: '忘记密码',
      ok: '确定',
      cancel: '取消',
      loginButtonText: '登录',
      tips: {
        reset1: '长按 NanoKVM 上的 BOOT 按键 10 秒钟来重置帐号。',
        reset2: '详细操作步骤可参考此文档：',
        reset3: '网页默认帐号：',
        reset4: 'SSH 默认帐号：',
        change1: '请注意，此操作将同时更新以下密码：',
        change2: '网页的登录密码',
        change3: '系统 root 用户的密码（SSH 登录密码）',
        change4: '如果您忘记了密码，需要长按 NanoKVM 上的 BOOT 按键来重置密码。'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: '配置 NanoKVM Wi-Fi 信息',
      success: '请前往设备检查 NanoKVM 的网络状态。',
      failed: '操作失败，请重试。',
      invalidMode: '当前模式不支持配置网络。请先前往设备启用 Wi-Fi 配置模式。',
      confirmBtn: '确定',
      finishBtn: '完成',
      ap: {
        authTitle: '身份验证',
        authDescription: '请输入 AP 密码以继续',
        authFailed: '密码错误',
        passPlaceholder: 'AP 密码',
        verifyBtn: '验证'
      }
    },
    screen: {
      scale: '缩放',
      title: '屏幕',
      video: '视频模式',
      videoDirectTips: '该模式需启用 HTTPS，请前往「设置 - 设备」中开启',
      resolution: '分辨率',
      auto: '自动',
      autoTips:
        '在某些分辨率下可能存在花屏或鼠标偏移的情况，请调整远程主机分辨率或者不使用自动模式。',
      fps: '帧率',
      customizeFps: '自定义',
      quality: '图像质量',
      qualityLossless: '无损',
      qualityHigh: '高',
      qualityMedium: '中',
      qualityLow: '低',
      frameDetect: '帧差检测',
      frameDetectTip: '计算帧之间的差异，当检测到远程主机画面不变时，停止传输视频流',
      resetHdmi: '重置 HDMI',
      mixedH264: {
        title: 'H.264 视频流冲突',
        description:
          '检测到 H.264 Direct 和 H.264 WebRTC 同时使用，可能导致画面撕裂或花屏。请只保留一种 H.264 模式。'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC 连接失败',
        description: '请检查网络连接或切换视频模式。'
      },
      captureStatus: {
        hdmiError: 'HDMI 画面异常',
        unsupportedResolution: '当前分辨率不支持',
        retrieving: '正在获取画面...',
        changingResolution: '正在切换分辨率...',
        updateFailed: '画面暂时无法更新',
        videoError: '视频显示异常',
        noHdmi: '未检测到 HDMI 信号',
        unavailable: '画面暂时无法显示'
      }
    },
    keyboard: {
      title: '键盘',
      paste: '粘贴',
      tips: '仅支持标准键盘的字母和符号',
      placeholder: '请输入内容',
      submit: '确定',
      virtual: '虚拟键盘',
      readClipboard: '从剪贴板读取',
      clipboardPermissionDenied: '剪贴板权限被拒绝。请允许您的浏览器访问剪贴板。',
      clipboardReadError: '无法读取剪贴板',
      dropdownEnglish: '英语',
      dropdownGerman: '德语',
      dropdownFrench: '法语',
      dropdownRussian: '俄语',
      shortcut: {
        title: '快捷键',
        custom: '自定义',
        capture: '点击此处捕获快捷键',
        clear: '清空',
        save: '保存',
        captureTips: '捕获系统级按键（如 Windows 键）需要全屏权限。',
        enterFullScreen: '切换全屏模式。'
      },
      leaderKey: {
        title: '引导键',
        desc: '绕过浏览器限制，向远程主机发送被系统拦截的快捷键。',
        howToUse: '使用方法',
        simultaneous: {
          title: '组合按键模式',
          desc1: '按住引导键不放，同时按下目标快捷键。',
          desc2: '操作直观，但少数快捷键可能因系统占用而无法生效。'
        },
        sequential: {
          title: '序列输入模式',
          desc1: '点击引导键开始 → 依次点击快捷键 → 再次点击引导键结束。',
          desc2: '步骤较多，但能完美避开系统键位冲突。'
        },
        enable: '启用引导键',
        tip: '设为引导键后，该按键将仅用于触发快捷键，不再作为普通按键使用。',
        placeholder: '请按下引导键',
        shiftRight: '右 Shift',
        ctrlRight: '右 Ctrl',
        metaRight: '右 Win',
        submit: '确定',
        recorder: {
          rec: '录制中',
          activate: '激活按键',
          input: '请按下快捷键...'
        }
      }
    },
    mouse: {
      title: '鼠标',
      cursor: '光标样式',
      default: '默认光标',
      pointer: '悬浮指针',
      cell: '单元指针',
      text: '文本指针',
      grab: '抓取指针',
      hide: '隐藏指针',
      mode: '鼠标模式',
      absolute: '绝对模式',
      relative: '相对模式',
      direction: '滚轮方向',
      scrollUp: '向上',
      scrollDown: '向下',
      speed: '滚轮速度',
      fast: '快',
      slow: '慢',
      requestPointer: '正在使用鼠标相对模式，请点击桌面获取鼠标指针。',
      resetHid: '重置 HID',
      hidOnly: {
        title: 'HID-Only 模式',
        desc: '若使用过程中遇到鼠标键盘无响应，且重置 HID 无效，可能是 NanoKVM 与您的设备存在兼容性问题。建议尝试启用 HID-Only 模式以提升兼容性。',
        tip1: '启用 HID-Only 模式会卸载虚拟 U 盘和虚拟网络',
        tip2: 'HID-Only 模式下，镜像挂载将被禁用',
        tip3: '切换模式后将自动重启 NanoKVM',
        enable: '启用 HID-Only 模式',
        disable: '关闭 HID-Only 模式'
      }
    },
    image: {
      title: '镜像',
      loading: '加载中',
      empty: '无镜像文件',
      mountMode: '挂载模式',
      mountFailed: '挂载失败',
      mountDesc: '在某些系统中，需要在远程主机中弹出虚拟硬盘后再挂载镜像。',
      unmountFailed: '卸载失败',
      unmountDesc: '在某些系统中，需要在远程主机中手动弹出后再卸载镜像。',
      refresh: '刷新镜像列表',
      attention: '注意',
      deleteConfirm: '确定要删除该镜像吗？',
      okBtn: '确定',
      cancelBtn: '取消',
      tips: {
        title: '如何上传',
        usb1: '将 NanoKVM 通过 USB 连接到你的电脑；',
        usb2: '确保已经挂载了虚拟硬盘（设置 - 虚拟硬盘）；',
        usb3: '在电脑上打开虚拟硬盘，将镜像文件拷贝到虚拟硬盘的根目录下。',
        scp1: '确保 NanoKVM 和你的电脑在同一个局域网内；',
        scp2: '在电脑上打开终端软件，使用 SCP 命令将镜像文件上传到 NanoKVM 的 /data 目录。',
        scp3: '示例：scp your-image-path root@your-nanokvm-ip:/data',
        tfCard: 'TF 卡',
        tf1: '该方式适用于 Linux 系统',
        tf2: '将 TF 卡从 NanoKVM 中取出（FULL 版本需要先拆开外壳）；',
        tf3: '将 TF 卡插入读卡器并连接到你的电脑；',
        tf4: '从电脑上拷贝镜像文件到 TF 卡的 /data 目录下；',
        tf5: '将 TF 卡重新插入 NanoKVM。'
      }
    },
    script: {
      title: '脚本',
      upload: '上传',
      run: '运行',
      runBackground: '后台运行',
      runFailed: '运行失败',
      attention: '注意',
      delDesc: '确定要删除该文件吗？',
      confirm: '确定',
      cancel: '取消',
      delete: '删除',
      close: '关闭'
    },
    terminal: {
      title: '终端',
      nanokvm: 'NanoKVM 终端',
      serial: '串口终端',
      serialPort: '串口',
      serialPortPlaceholder: '请输入串口',
      baudrate: '波特率',
      parity: '奇偶校验',
      parityNone: '无',
      parityEven: '偶校验',
      parityOdd: '奇校验',
      flowControl: '流量控制',
      flowControlNone: '无',
      flowControlSoft: '软件',
      flowControlHard: '硬件',
      dataBits: '数据位',
      stopBits: '停止位',
      confirm: '确定'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: '指令发送中...',
      sent: '指令已发送',
      input: '请输入MAC地址',
      ok: '确定'
    },
    download: {
      title: '下载镜像',
      input: '请输入远程镜像 URL',
      ok: '确定',
      disabled: '/data 是只读分区，无法下载镜像',
      uploadbox: '将文件拖放到此处或单击选择',
      inputfile: '请输入镜像文件',
      NoISO: '无 ISO',
      sha256: 'SHA-256（可选）',
      sha256Placeholder: '请输入 64 位 SHA-256 校验和',
      invalidSHA256: 'SHA-256 必须是 64 位十六进制字符串',
      failed: '下载失败',
      success: '下载成功',
      checksumFailed: '下载失败：SHA-256 校验失败',
      cancel: '取消',
      cancelFailed: '取消下载失败'
    },
    power: {
      title: '电源',
      showConfirm: '显示确认框',
      showConfirmTip: '电源操作需要二次确认',
      reset: '重启',
      power: '电源',
      powerShort: '电源（短按）',
      powerLong: '电源（长按）',
      resetConfirm: '确认执行重启操作吗？',
      powerConfirm: '确认执行电源操作吗？',
      okBtn: '确认',
      cancelBtn: '取消'
    },
    settings: {
      title: '设置',
      display: {
        title: '显示',
        loading: '加载中...',
        active: '当前 EDID',
        activeUnknown: 'NanoKVM 自启动以来未写入过 EDID，因此主机看到的显示器标识未知。',
        appliedAt: '应用于 {{time}}',
        download: '下载',
        downloadBackup: '下载上一个',
        preset: '显示器预设',
        presetPlaceholder: '选择显示器',
        upload: '上传',
        selected: '已选择的 EDID',
        errors: '错误',
        warnings: '警告',
        info: '信息',
        unknownMonitor: '未知显示器',
        edidVersion: 'EDID {{version}}',
        audioYes: '支持音频',
        audioNo: '无音频',
        extensionBlocks: '扩展块：{{blocks}}',
        apply: '应用',
        applyTitle: '应用此 EDID？',
        before: '当前',
        after: '新的',
        hdmiNotice: '写入 EDID 期间视频采集会中断，完成后会自动恢复。',
        powerCycleNotice: '必须将本设备断开电源并重新接通，新的 EDID 才会生效。',
        applied: 'EDID 已应用并校验通过。',
        applyFailed: 'EDID 应用失败。',
        busy: '视频芯片正忙，请重试。',
        unsupported: '本设备不支持修改 EDID。',
        toolMissing: '此固件中缺少 EDID 工具。',
        noAudio: '此 EDID 未声明音频，主机可能会停止输出声音。',
        oldVersion: '此 EDID 使用低于 1.4 的版本。',
        interlaced: '首选时序为隔行扫描。',
        tooLarge: '首选时序高于 1920x1080 60 Hz，超出 NanoKVM 的采集能力。',
        recovery: '恢复',
        recoveryNeeded:
          '上一次写入未通过校验，视频芯片的 EDID 区域处于未知状态。请恢复出厂 EDID，使其重新回到已知状态。',
        recoveryDesc: '当已应用的 EDID 导致主机无画面时，将已知的 EDID 重新写回视频芯片。',
        restoreFactory: '恢复出厂 EDID',
        restoreBackup: '恢复上一个 EDID',
        restoreTitle: '恢复此 EDID？',
        restoreFactoryTarget: 'NanoKVM 出厂自带的 EDID。',
        restoreBackupTarget: '最近一次备份，应用于 {{time}}。',
        restoreNotice: '恢复与应用采用相同的写入方式，后果也相同。',
        restored: 'EDID 已恢复并校验通过。',
        restoreFailed: 'EDID 恢复失败。',
        mismatchTitle: '写入内容与读回内容',
        mismatchWritten: '已写入',
        mismatchRead: '读回',
        restoreOkBtn: '恢复',
        hardware: '检测到的硬件：{{hardware}}',
        hardwareUnknown: '未知',
        confirmWord: '应用',
        confirmPrompt: '输入 {{word}} 以启用应用按钮。',
        okBtn: '应用',
        cancelBtn: '取消'
      },
      passthrough: {
        title: 'USB 透传',
        loading: '加载中...',
        hidWarning: '启动透传会让出键盘、鼠标和虚拟媒体',
        hidWarningDesc:
          'NanoKVM 只有一个 USB 设备控制器，而代理需要独占它。因此会话运行期间，远程主机看到的是被透传的设备，而不是 NanoKVM 的键盘、鼠标和虚拟媒体。会话一停止，它们会自动恢复。此网页界面不受影响，您随时可以在本页停止会话。',
        isoWarning: '摄像头、麦克风等同步传输设备无法透传',
        isoWarningDesc:
          '该硬件只能承载控制传输、批量传输和中断传输。音频和视频设备无论如何绑定都无法工作。',
        session: '会话',
        activeDesc: '已导入一台设备，代理正占用 USB 控制器。',
        inactiveDesc: '当前没有会话。键盘、鼠标和虚拟媒体工作正常。',
        device: '设备',
        busId: '总线 ID',
        speed: '速度',
        exporter: '导出端',
        local: '导入为',
        localValue: '总线 {{bus}}，地址 {{address}}',
        udc: 'USB 控制器',
        pid: '代理进程号',
        startedAt: '开始时间',
        isoDevice: '该设备声明为音频或视频类，需要同步传输，因此无法工作。',
        exporterLabel: '导出端地址',
        exporterHint: 'NanoKVM 连接的主机和端口。使用下面的隧道时即为 {{exporter}}。',
        busIdLabel: '本机上的总线 ID',
        busIdHint: 'usbip list -l 为该设备打印的 busid，例如 {{example}}。',
        start: '启动透传',
        stop: '停止透传',
        startTitle: '要启动 USB 透传吗？',
        startDevice: 'NanoKVM 将从 {{exporter}} 导入 {{busId}}。',
        startHid: '会话运行期间，USB 键盘、鼠标和虚拟媒体将停止工作；停止会话后会自动恢复。',
        startIso: '摄像头、麦克风等同步传输设备在该硬件上无法工作。',
        startWeb: '此网页界面仍可使用，您随时可以在本页停止会话。',
        okBtn: '启动',
        cancelBtn: '取消',
        instructions: '在您自己的电脑上',
        instructionsDesc:
          '按照设计，无需安装任何客户端代理。请在拥有该设备的电脑上运行以下标准 usbip 命令。',
        copyFailed: '复制失败，请手动复制命令。',
        directNote:
          '不使用隧道时，usbipd 必须在您的网络上可达，并且上面的导出端地址要指向它。usbip 以明文传输设备数据，建议优先使用隧道。',
        steps: {
          modprobe: {
            title: '加载导出端驱动',
            desc: 'usbip-host 让内核可以把本地设备交出去，默认不会加载。'
          },
          list: {
            title: '找到设备',
            desc: '列出所有本地设备及其 busid 和厂商:产品编号。记下要透传的那台设备的 busid。'
          },
          bind: {
            title: '绑定到 usbip',
            desc: '把设备从原有驱动上摘下，在解除绑定之前它在本机上将不可用。'
          },
          serve: {
            title: '提供服务',
            desc: 'usbipd 会在前台运行，等待 NanoKVM 导入该设备。',
            notice:
              '标准 usbipd 没有监听地址选项，会监听所有网卡。请在防火墙上关闭端口 {{port}}，只允许下面的隧道访问。'
          },
          tunnel: {
            title: '指向 NanoKVM',
            desc: 'SSH 反向隧道：NanoKVM 本地回环上的 {{port}} 端口即为本机的 usbipd。整个会话期间请保持它运行。'
          },
          exporter: {
            title: '用它作为导出端',
            desc: '把它填入上面的导出端地址，输入总线 ID，然后启动会话。'
          },
          unbind: {
            title: '归还设备',
            desc: '会话停止后，用它把设备交还给本机原有的驱动。'
          }
        }
      },
      mcp: {
        title: 'MCP 服务',
        service: '远程控制 MCP',
        serviceDesc: '允许可信的 MCP 客户端控制键盘、鼠标并获取屏幕截图',
        securityWarning:
          '任何持有此 API key 的人都可以控制远程主机并查看屏幕，请使用 HTTPS，并仅在可信网络中启用。',
        endpoint: '服务地址',
        apiKey: 'API Key',
        regenerateConfirmTitle: '重新生成 MCP API key？',
        regenerateConfirmDesc: '当前 key 将立即失效。',
        enableConfirmTitle: '启用外部 MCP 控制？',
        enableConfirmDesc: '启用 MCP 将停止 PicoClaw，并关闭当前活动的 PicoClaw 会话。',
        failed: 'MCP 操作失败',
        copyFailed: '复制失败，请手动复制。',
        okBtn: '确认',
        cancelBtn: '取消'
      },
      about: {
        title: '关于 NanoKVM',
        information: '信息',
        ip: 'IP',
        mdns: 'mDNS',
        application: '应用版本',
        applicationTip: 'NanoKVM 网页应用版本',
        image: '镜像版本',
        imageTip: 'NanoKVM 系统镜像版本',
        deviceKey: '设备码',
        community: '社区',
        hostname: '主机名',
        hostnameUpdated: '主机名修改成功，重启后生效',
        ipType: {
          Wired: '有线',
          Wireless: '无线',
          Other: '其他'
        }
      },
      appearance: {
        title: '外观',
        display: '显示',
        language: '语言',
        languageDesc: '选择界面语言',
        webTitle: '网页标题',
        webTitleDesc: '自定义网站页面标题',
        menuBar: {
          title: '菜单栏',
          mode: '显示方式',
          modeDesc: '菜单栏在屏幕上的显示方式',
          modeOff: '关闭',
          modeAuto: '自动隐藏',
          modeAlways: '始终显示',
          keyboardLedStatus: '键盘锁定状态指示灯',
          keyboardLedStatusDesc: '显示远程主机的 Num Lock、Caps Lock 和 Scroll Lock 状态',
          icons: '菜单图标',
          iconsDesc: '是否在菜单栏中显示子菜单图标'
        }
      },
      keyboardLedStatus: {
        groupLabel: '远程键盘锁定状态',
        indicatorLabel: '{{label}}：{{state}}',
        numLock: '数字锁定',
        numLockShort: '数',
        capsLock: '大写锁定',
        capsLockShort: '大',
        scrollLock: '滚动锁定',
        scrollLockShort: '滚',
        on: '开启',
        off: '关闭',
        unknown: '未知'
      },
      device: {
        title: '设备',
        oled: {
          title: 'OLED',
          description: '设置 OLED 屏幕自动休眠时间',
          0: '永不',
          15: '15秒',
          30: '30秒',
          60: '1分钟',
          180: '3分钟',
          300: '5分钟',
          600: '10分钟',
          1800: '30分钟',
          3600: '1小时'
        },
        ssh: {
          description: '启用 SSH 远程访问',
          tip: '启用前请务必设置强密码（帐号 - 修改密码）'
        },
        advanced: '高级设置',
        swap: {
          title: '交换',
          disable: '禁用',
          description: '设置交换文件大小',
          tip: '启用该功能可能会减少SD卡使用寿命！'
        },
        mouseJiggler: {
          title: '鼠标抖动',
          description: '防止远程主机休眠',
          disable: '关闭',
          absolute: '绝对模式',
          relative: '相对模式'
        },
        mdns: {
          description: '启用 mDNS 发现服务',
          tip: '如果您未使用此功能，建议将其关闭'
        },
        hdmi: {
          description: '启用 HDMI/显示器 输出功能',
          idleTimeoutTitle: '无观看者自动停止采集',
          idleTimeoutDescription: '没有活跃观看者后停止 HDMI 采集，0 表示永不停止',
          minutes: '分钟'
        },
        autostart: {
          title: '自动启动脚本设置',
          description: '管理能够在 NanoKVM 启动时自动运行的脚本文件',
          new: '创建新脚本',
          deleteConfirm: '确定要删除该文件吗？',
          yes: '是',
          no: '否',
          scriptName: '自动启动脚本名称',
          scriptContent: '自动启动脚本内容',
          settings: '设置'
        },
        hidOnly: 'HID-Only 模式',
        hidOnlyDesc: '该模式下不再挂载虚拟设备，仅保留基础的 HID 控制功能。',
        disk: '虚拟U盘',
        diskDesc: '在远程主机中挂载虚拟U盘',
        network: '虚拟网卡',
        networkDesc: '在远程主机中挂载虚拟网卡',
        networkProtocol: '网络协议',
        networkProtocolDesc: '新主机使用 NCM，旧版 Windows 使用 RNDIS',
        reboot: '重新启动',
        rebootDesc: '您确定要重新启动 NanoKVM 吗？',
        okBtn: '是',
        cancelBtn: '否'
      },
      network: {
        title: '网络',
        wifi: {
          title: 'Wi-Fi',
          description: '配置 Wi-Fi 信息',
          apMode: 'AP 模式已启用，请扫描二维码连接 Wi-Fi',
          connect: '连接 Wi-Fi',
          connectDesc1: '请输入网络名称和密码',
          connectDesc2: '请输入密码以连接此网络',
          disconnect: '是否要断开该网络连接？',
          failed: '连接失败，请重试',
          ssid: '名称',
          password: '密码',
          joinBtn: '加入',
          confirmBtn: '确定',
          cancelBtn: '取消'
        },
        tls: {
          description: '启用 HTTPS 协议',
          tip: '注意：使用 HTTPS 可能导致延迟增加，特别是在 MJPEG 视频模式下。'
        },
        bridge: {
          title: '网络桥接',
          twoDevices: '路由器会将 NanoKVM 和被控电脑视为两台独立设备，各自拥有自己的地址。',
          loading: '加载中...',
          state: '状态',
          states: {
            disabled: '已禁用',
            enabled: '已启用',
            rolledBack: '已回滚',
            failed: '失败',
            pending: '进行中'
          },
          uplink: '上行接口',
          ports: '端口',
          protocol: '设备协议',
          up: '已连接',
          down: '未连接',
          enableTitle: '启用网络桥接？',
          disableTitle: '禁用网络桥接？',
          reconnect: '地址迁移期间，管理连接会短暂断开并重新连接。',
          rollback: '如果验证失败，将自动恢复到之前的网络配置。',
          enableBtn: '启用',
          disableBtn: '禁用',
          cancelBtn: '取消',
          interrupted: '应用过程中连接中断，正在重新检查当前状态。',
          pendingNotice: '桥接变更仍在进行中，或在完成前被中断。',
          revert: '恢复之前的配置',
          rolledBackNotice: '上次变更已回滚，之前的网络配置已恢复。',
          verifyFailed: '验证失败：{{gates}}',
          gates: {
            address: '地址',
            gateway: '网关',
            inbound: '入站连接'
          },
          inboundWeak:
            '入站检查仅通过 NanoKVM 自连接完成。这只能证明 Web 服务正在监听且本机可达，并不能证明来自网络的请求可以到达。',
          failedNotice: '上次变更未能撤销。可能只能通过 Wi-Fi AP 或串口控制台访问 NanoKVM。'
        },
        dns: {
          title: 'DNS',
          description: '配置 NanoKVM 使用的 DNS 服务器',
          mode: '模式',
          dhcp: 'DHCP',
          manual: '手动',
          add: '添加 DNS',
          save: '保存',
          invalid: '请输入有效的 IP 地址',
          noDhcp: '当前未获取到 DHCP DNS',
          saved: 'DNS 设置已保存',
          saveFailed: '保存 DNS 设置失败',
          unsaved: '有未保存的更改',
          maxServers: '最多允许 {{count}} 个 DNS 服务器',
          dnsServers: 'DNS 服务器',
          dhcpServersDescription: 'DNS 服务器由 DHCP 自动获取',
          manualServersDescription: 'DNS 服务器可以手动编辑',
          networkDetails: '网络详情',
          interface: '接口',
          ipAddress: 'IP 地址',
          subnetMask: '子网掩码',
          router: '路由器',
          none: '无'
        }
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: '内存优化',
          tip: '当内存占用超过限制时，会更积极地执行垃圾回收来尝试释放内存。需重启 Tailscale 后生效。'
        },
        swap: {
          title: '交换内存',
          tip: '如果启用内存优化后依然存在问题，可以尝试开启交换内存。启用后会将交换文件设置为256MB，可以在「设置 - 设备」中修改该选项。'
        },
        restart: '取定要重启 Tailscale 吗？',
        stop: '确定要停止 Tailscale 吗？',
        stopDesc: '退出 Tailscale 并禁用开机自动启动。',
        loading: '加载中...',
        notInstall: '未检测到 Tailscale，请先安装',
        install: '安装',
        installing: '安装中',
        failed: '安装失败',
        retry: '请刷新后重试，或尝试手动安装',
        download: '下载',
        package: '安装包',
        unzip: '并解压',
        upTailscale: '将 tailscale 上传到 NanoKVM 的 /usr/bin/ 目录',
        upTailscaled: '将 tailscaled 上传到 NanoKVM 的 /usr/sbin/ 目录',
        refresh: '刷新页面',
        notRunning: 'Tailscale 尚未运行，请先执行启动操作',
        run: '启动',
        notLogin: '该设备尚未绑定，请点击登录并将这台设备绑定到您的账号。',
        urlPeriod: '该链接10分钟内有效',
        login: '登录',
        loginSuccess: '登录完成',
        enable: '启用 Tailscale',
        deviceName: '设备名称',
        deviceIP: '设备地址',
        account: '账号',
        logout: '退出',
        logoutDesc: '确定要退出吗？',
        uninstall: '卸载 Tailscale',
        uninstallDesc: '确定要卸载 Tailscale 吗？',
        okBtn: '确认',
        cancelBtn: '取消'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: '加载中...',
        notInstall: '未安装',
        notConfigured: '未配置',
        stopped: '已停止',
        running: '运行中',
        connected: '已连接',
        error: '错误',
        arguments: '启动参数',
        argumentsTip: '启动服务时传入的命令行参数。',
        env: '环境变量',
        envKey: '名称',
        envValue: '值',
        envAdd: '添加变量',
        envRemove: '移除',
        configured: '已配置',
        save: '保存',
        saved: '配置已保存',
        start: '启动',
        stop: '停止',
        restart: '重启',
        logs: '日志',
        logsEmpty: '暂无日志',
        refresh: '刷新',
        binary: '可执行文件',
        binaryShipped: '固件自带',
        binaryCustom: '自行上传',
        binaryUpload: '上传可执行文件',
        binaryRevert: '恢复固件自带版本',
        binaryRevertDesc: '确定要删除已上传的可执行文件并恢复固件自带的版本吗？',
        serverWarning: '未加限制的服务器相当于开放代理',
        noHealthSignal:
          '该服务不提供健康状态信号，因此 NanoKVM 只能确认进程正在运行，无法确认隧道是否已连接。',
        memoryWarning: '同时运行多个远程访问服务可能会耗尽内存',
        okBtn: '确定',
        cancelBtn: '取消'
      },
      update: {
        title: '检查更新',
        queryFailed: '获取版本号失败',
        updateFailed: '更新失败，请重试',
        isLatest: '已经是最新版本。',
        available: '有新的可用版本，确定要更新吗？',
        updating: '更新中，请稍候...',
        confirm: '确定',
        cancel: '取消',
        preview: '预览更新',
        previewDesc: '率先体验即将推出的新功能和优化',
        previewTip: '预览版更新可能包含一些不稳定因素或未完善的功能！',
        customServer: {
          title: '自定义更新服务器',
          desc: '从指定服务器检查并下载在线更新',
          invalidUrl:
            '请输入有效的 HTTP 或 HTTPS 服务器目录，不能包含查询参数、片段或 latest.json。',
          loadFailed: '读取更新服务器配置失败。',
          saveFailed: '保存更新服务器配置失败。',
          saved: '更新服务器配置已保存。',
          save: '保存',
          confirmTitle: '使用自定义更新服务器？',
          confirmDesc:
            'SHA-512 只能验证安装包与该服务器提供的清单一致，不能证明安装包来自 NanoKVM 官方。错误或恶意的服务器可能导致设备不可用、数据丢失或系统被接管。',
          confirm: '仍然使用',
          previewDisabled: '启用自定义更新服务器时，预览更新不可用'
        },
        offline: {
          title: '离线更新',
          desc: '通过本地安装包进行更新',
          upload: '上传',
          checksumPlaceholder: 'SHA-256 校验和（可选）',
          invalidChecksum: 'SHA-256 校验和必须为 64 位十六进制字符。',
          checksumMismatch: 'SHA-256 校验失败，安装包可能已损坏。',
          invalidName: '文件名格式错误，请前往 GitHub 发布页下载安装包。',
          updateFailed: '更新失败，请重试'
        }
      },
      account: {
        title: '帐号',
        webAccount: '网页帐号',
        role: '角色',
        roles: {
          admin: '管理员',
          user: '普通用户'
        },
        password: '密码',
        updateBtn: '修改',
        logoutBtn: '退出',
        logoutDesc: '确定要退出吗？',
        okBtn: '确定',
        cancelBtn: '取消',
        users: {
          title: '用户管理',
          create: '创建用户',
          enabled: '已启用',
          disabled: '已禁用',
          deviceOwner: '设备所有者',
          resetPassword: '重置密码',
          delete: '删除',
          deleteConfirm: '删除此用户并撤销其全部会话？',
          created: '用户已创建',
          deleted: '用户已删除',
          passwordUpdated: '密码已更新',
          loadFailed: '加载用户失败',
          saveFailed: '保存用户失败',
          deleteFailed: '删除用户失败'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw 助手',
      empty: '打开面板并发送任务后，PicoClaw 会开始工作。',
      inputPlaceholder: '描述你希望 PicoClaw 执行的操作',
      newConversation: '新对话',
      processing: '正在处理中...',
      agent: {
        defaultTitle: '通用助手',
        defaultDescription: '适合普通问答、搜索和工作区辅助。',
        kvmTitle: '远程控制',
        kvmDescription: '通过 NanoKVM 操作下游远程主机。',
        switched: '角色已切换',
        switchFailed: '切换角色失败'
      },
      send: '发送',
      cancel: '取消',
      status: {
        connecting: '正在连接 Gateway...',
        connected: 'PicoClaw 会话已连接',
        disconnected: 'PicoClaw 会话已关闭',
        stopped: '已发送停止请求',
        runtimeStarted: 'PicoClaw 运行时已启动',
        runtimeStartFailed: '启动 PicoClaw 运行时失败',
        runtimeStopped: 'PicoClaw 运行时已停止',
        runtimeStopFailed: '停止 PicoClaw 运行时失败',
        controlSwitchedToMCP: '控制权已切换到外部 MCP 服务'
      },
      connection: {
        runtime: {
          checking: '检查中',
          restoring: '正在恢复 PicoClaw',
          ready: '运行时已就绪',
          stopped: '运行时未启动',
          blockedByMCP: '外部 MCP 控制已启用',
          readyBlockedByMCP: '运行时正在运行，但外部 MCP 当前控制设备输入。',
          readyWithoutControl: '运行时正在运行，请先授予 PicoClaw 设备控制权后再重新连接。',
          unavailable: '运行时不可用',
          configError: '配置错误'
        },
        transport: {
          connecting: '连接中',
          connected: '已连接',
          disconnected: '未连接',
          reconnect: '重新连接',
          reconnectDescription: '重新连接到正在运行的 PicoClaw 会话。',
          reconnectBlocked: 'PicoClaw 需要先获得设备控制权才能重新连接。'
        },
        run: {
          idle: '空闲',
          busy: '执行中'
        }
      },
      message: {
        toolAction: '动作',
        observation: '观察',
        screenshot: '截图'
      },
      overlay: {
        locked: 'PicoClaw 正在控制设备，手动输入已暂停。'
      },
      control: {
        picoclaw: '设备控制：PicoClaw',
        picoclawDescription: 'PicoClaw 可以写入键鼠，手动输入可能会被暂停。',
        mcp: '设备控制：外部 MCP',
        mcpDescription: '外部 MCP 可以写入设备，PicoClaw 不会接管键鼠。',
        off: '设备控制：手动/无 AI',
        offDescription: 'AI 不会写入键鼠，手动控制保持可用。',
        transitioning: '设备控制：正在切换',
        transitioningDescription: '正在同步设备控制权，请稍候。',
        grant: '接管设备',
        release: '交还控制',
        releasing: '正在释放...',
        switching: '正在切换...',
        releasingLabel: '设备控制：正在释放',
        releasingDescription: '正在交还设备控制，PicoClaw 已停止当前写入。',
        granted: '已授予 PicoClaw 控制权',
        released: '已交还设备控制权',
        grantFailed: '授予 PicoClaw 控制权失败',
        releaseFailed: '释放 PicoClaw 控制权失败',
        grantConfirmTitle: '将设备控制切换到 PicoClaw？',
        grantConfirmDesc: '外部 MCP 的设备写入将被中断。'
      },
      install: {
        install: '安装 PicoClaw',
        installing: '正在安装 PicoClaw',
        success: 'PicoClaw 安装成功',
        failed: 'PicoClaw 安装失败',
        uninstalling: '正在卸载运行时...',
        uninstalled: '运行时卸载成功。',
        uninstallFailed: '卸载失败。',
        requiredTitle: '未安装 PicoClaw',
        requiredDescription: '在启动 PicoClaw 运行时之前，需要先下载安装 PicoClaw。',
        progressDescription: '正在下载并安装 PicoClaw。',
        stages: {
          preparing: '准备中',
          downloading: '下载中',
          extracting: '解压中',
          verifying: '验证中',
          installing: '安装中',
          installed: '已安装',
          install_timeout: '已超时',
          install_failed: '失败'
        }
      },
      model: {
        requiredTitle: '需要配置模型',
        requiredDescription: '在使用 PicoClaw 聊天之前，请先配置 PicoClaw 模型。',
        docsTitle: '配置指南',
        docsDesc: '支持的模型与协议',
        menuLabel: '配置模型',
        modelIdentifier: '模型标识',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API Key',
        apiKeyPlaceholder: '请输入模型 API Key',
        save: '保存模型配置',
        saving: '正在保存模型配置',
        saved: '模型配置已保存',
        saveFailed: '保存模型配置失败',
        invalid: '模型标识、API Base URL 和 API Key 不能为空'
      },
      uninstall: {
        menuLabel: '卸载',
        confirmTitle: '卸载 PicoClaw',
        confirmContent: '确定要卸载 PicoClaw 吗？这会删除可执行文件和所有配置文件。',
        confirmOk: '卸载',
        confirmCancel: '取消'
      },
      history: {
        title: '历史会话',
        loading: '正在加载历史会话...',
        emptyTitle: '还没有历史会话',
        emptyDescription: '之前的 PicoClaw 会话会显示在这里。',
        loadFailed: '加载会话历史失败',
        deleteFailed: '删除会话失败',
        deleteConfirmTitle: '删除会话',
        deleteConfirmContent: '确定要删除“{{title}}”吗？',
        deleteConfirmOk: '删除',
        deleteConfirmCancel: '取消',
        messageCount_one: '{{count}} 条消息',
        messageCount_other: '{{count}} 条消息',
        messageCount: '{{count}} 条消息'
      },
      config: {
        startRuntime: '启动 PicoClaw',
        stopRuntime: '停止 PicoClaw'
      },
      start: {
        enableConfirmTitle: '切换为 PicoClaw 控制？',
        enableConfirmDesc: '启动 PicoClaw 前会中断外部 MCP 的设备写入。',
        enableConfirmOk: '启动 PicoClaw',
        enableConfirmCancel: '取消',
        title: '启动 PicoClaw',
        description: '启动运行时后即可开始使用 PicoClaw 助手。',
        switchFromMCP: '切换到 PicoClaw 并启动',
        takeoverAndStart: '接管并启动'
      }
    },
    error: {
      title: '我们遇到了问题',
      refresh: '刷新'
    },
    fullscreen: {
      toggle: '切换全屏'
    },
    menu: {
      collapse: '收起',
      expand: '展开'
    }
  }
};

export default zh;
