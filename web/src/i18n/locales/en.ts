const en = {
  translation: {
    head: {
      desktop: 'Remote Desktop',
      login: 'Login',
      changePassword: 'Change Password',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Login',
      placeholderUsername: 'Username',
      placeholderPassword: 'Password',
      placeholderCurrentPassword: 'Current password',
      placeholderPassword2: 'Please enter password again',
      noEmptyUsername: 'Username required',
      noEmptyPassword: 'Password required',
      passwordLength: 'Password must be between 8 and 72 characters',
      noAccount: 'Failed to get user information, please refresh web page or reset password',
      invalidUser: 'Invalid username or password',
      locked: 'Too many logins, please try again later',
      globalLocked: 'System under protection, please try again later',
      error: 'Unexpected error',
      invalidCurrentPassword: 'Current password is incorrect',
      changePassword: 'Change Password',
      changePasswordDesc: 'For the security of your device, please change the password!',
      differentPassword: 'Passwords do not match',
      illegalUsername: 'Username contains illegal characters',
      illegalPassword: 'Password contains illegal characters',
      forgetPassword: 'Forgot Password',
      ok: 'Ok',
      cancel: 'Cancel',
      loginButtonText: 'Login',
      tips: {
        reset1:
          'To reset the passwords, press and hold the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'For detailed steps, please consult this document:',
        reset3: 'Web default account:',
        reset4: 'SSH default account:',
        change1: 'Please note that this action will change the following passwords:',
        change2: 'Web login password',
        change3: 'System root password (SSH login password)',
        change4: 'To reset the passwords, press and hold the BOOT button on the NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Configure Wi-Fi for NanoKVM',
      success: 'Please go to the device to check the network status of NanoKVM.',
      failed: 'Operation failed, please try again.',
      invalidMode:
        'The current mode does not support network setup. Please go to your device and enable Wi-Fi configuration mode.',
      confirmBtn: 'Ok',
      finishBtn: 'Finished',
      ap: {
        authTitle: 'Authentication Required',
        authDescription: 'Please enter the AP password to continue',
        authFailed: 'Invalid AP password',
        passPlaceholder: 'AP password',
        verifyBtn: 'Verify'
      }
    },
    screen: {
      scale: 'Scale',
      title: 'Screen',
      video: 'Video Mode',
      videoDirectTips: 'Enable HTTPS in "Settings > Device" to use this mode',
      resolution: 'Resolution',
      auto: 'Automatic',
      autoTips:
        "Screen tearing or mouse offset may occur at specific resolutions. Consider adjusting the remote host's resolution or disable automatic mode.",
      fps: 'FPS',
      customizeFps: 'Customize',
      quality: 'Quality',
      qualityLossless: 'Lossless',
      qualityHigh: 'High',
      qualityMedium: 'Medium',
      qualityLow: 'Low',
      frameDetect: 'Frame Detect',
      frameDetectTip:
        "Calculate the difference between frames. Stop transmitting video stream when no changes are detected on the remote host's screen.",
      resetHdmi: 'Reset HDMI',
      mixedH264: {
        title: 'H.264 stream conflict',
        description:
          'H.264 Direct and H.264 WebRTC are being used at the same time. This may cause screen tearing or corrupted video. Please use only one H.264 mode.'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC connection failed',
        description: 'Check the network connection or switch the video mode.'
      },
      captureStatus: {
        hdmiError: 'HDMI screen error',
        unsupportedResolution: 'Current resolution is not supported',
        retrieving: 'Getting screen...',
        changingResolution: 'Switching resolution...',
        updateFailed: 'Screen cannot update right now',
        videoError: 'Video display error',
        noHdmi: 'No HDMI signal detected',
        unavailable: 'Screen cannot be displayed right now'
      }
    },
    keyboard: {
      title: 'Keyboard',
      paste: 'Paste',
      tips: 'Only standard keyboard letters and symbols are supported',
      placeholder: 'Please input',
      submit: 'Submit',
      virtual: 'Keyboard',
      readClipboard: 'Read from Clipboard',
      clipboardPermissionDenied:
        'Clipboard permission denied. Please allow clipboard access in your browser.',
      clipboardReadError: 'Failed to read clipboard',
      dropdownEnglish: 'English',
      dropdownGerman: 'German',
      dropdownFrench: 'French',
      dropdownRussian: 'Russian',
      shortcut: {
        title: 'Shortcuts',
        custom: 'Custom',
        capture: 'Click here to capture shortcut',
        clear: 'Clear',
        save: 'Save',
        captureTips:
          'Capturing system-level keys (such as the Windows key) requires full-screen permission.',
        enterFullScreen: 'Toggle full-screen mode.'
      },
      leaderKey: {
        title: 'Leader Key',
        desc: 'Bypass browser restrictions and send system shortcuts directly to the remote host.',
        howToUse: 'How to Use',
        simultaneous: {
          title: 'Simultaneous Mode',
          desc1: 'Press and hold the Leader Key, then press the shortcut.',
          desc2: 'Intuitive, but may conflict with system shortcuts.'
        },
        sequential: {
          title: 'Sequential Mode',
          desc1:
            'Press the Leader Key → press the shortcut in sequence → press the Leader Key again.',
          desc2: 'Requires more steps, but completely avoids system conflicts.'
        },
        enable: 'Enable Leader Key',
        tip: 'When assigned as a Leader Key, this key functions exclusively as a shortcut trigger and loses its default behavior.',
        placeholder: 'Please press the Leader Key',
        shiftRight: 'Right Shift',
        ctrlRight: 'Right Ctrl',
        metaRight: 'Right Win',
        submit: 'Submit',
        recorder: {
          rec: 'REC',
          activate: 'Activate keys',
          input: 'Please press the shortcut...'
        }
      }
    },
    mouse: {
      title: 'Mouse',
      cursor: 'Cursor style',
      default: 'Default cursor',
      pointer: 'Pointer cursor',
      cell: 'Cell cursor',
      text: 'Text cursor',
      grab: 'Grab cursor',
      hide: 'Hide cursor',
      mode: 'Mouse mode',
      absolute: 'Absolute mode',
      relative: 'Relative mode',
      direction: 'Wheel direction',
      scrollUp: 'Scroll up',
      scrollDown: 'Scroll down',
      speed: 'Wheel speed',
      fast: 'Fast',
      slow: 'Slow',
      requestPointer: 'Using relative mode. Please click desktop to get mouse pointer.',
      resetHid: 'Reset HID',
      hidOnly: {
        title: 'HID-Only mode',
        desc: "If your mouse and keyboard stop responding and resetting HID doesn't help, it could be a compatibility issue between the NanoKVM and the device. Try to enable HID-Only mode for better compatibility.",
        tip1: 'Enabling HID-Only mode will unmount the virtual U-disk and virtual network',
        tip2: 'In HID-Only mode, image mounting is disabled',
        tip3: 'NanoKVM will automatically reboot after switching modes',
        enable: 'Enable HID-Only mode',
        disable: 'Disable HID-Only mode'
      }
    },
    image: {
      title: 'Images',
      loading: 'Loading...',
      empty: 'Nothing Found',
      mountMode: 'Mount mode',
      mountFailed: 'Mount failed',
      mountDesc:
        'On some systems, you need to eject the virtual disk from the remote host before mounting the image.',
      unmountFailed: 'Unmount failed',
      unmountDesc:
        'On some systems, you need to manually eject from the remote host before unmounting the image.',
      refresh: 'Refresh the image list',
      attention: 'Attention',
      deleteConfirm: 'Are you sure you want to delete this image?',
      okBtn: 'Yes',
      cancelBtn: 'No',
      tips: {
        title: 'How to upload',
        usb1: 'Connect the NanoKVM to your computer via USB.',
        usb2: 'Ensure that the virtual disk is mounted (Settings - Virtual Disk).',
        usb3: 'Open the virtual disk on your computer and copy the image file to the root directory of the virtual disk.',
        scp1: 'Make sure the NanoKVM and your computer are on the same local network.',
        scp2: 'Open a terminal on your computer and use the SCP command to upload the image file to the /data directory on the NanoKVM.',
        scp3: 'Example: scp your-image-path root@your-nanokvm-ip:/data',
        tfCard: 'TF Card',
        tf1: 'This method is supported on Linux system',
        tf2: 'Get TF card from the NanoKVM (for the FULL version, disassemble the case first).',
        tf3: 'Insert the TF card into a card reader and connect it to your computer.',
        tf4: 'Copy the image file to the /data directory on the TF card.',
        tf5: 'Insert the TF card into the NanoKVM.'
      }
    },
    script: {
      title: 'Scripts',
      upload: 'Upload',
      run: 'Run',
      runBackground: 'Run Background',
      runFailed: 'Run failed',
      attention: 'Attention',
      delDesc: 'Are you sure you want to delete this file?',
      confirm: 'Yes',
      cancel: 'No',
      delete: 'Delete',
      close: 'Close'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM Terminal',
      serial: 'Serial Port Terminal',
      serialPort: 'Serial Port',
      serialPortPlaceholder: 'Please enter the serial port',
      baudrate: 'Baud rate',
      parity: 'Parity',
      parityNone: 'None',
      parityEven: 'Even',
      parityOdd: 'Odd',
      flowControl: 'Flow control',
      flowControlNone: 'None',
      flowControlSoft: 'Soft',
      flowControlHard: 'Hard',
      dataBits: 'Data bits',
      stopBits: 'Stop bits',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Sending command...',
      sent: 'Command sent',
      input: 'Please enter the MAC',
      ok: 'Ok'
    },
    download: {
      title: 'Image Downloader',
      input: 'Please enter a remote image URL',
      ok: 'Ok',
      disabled: '/data partition is RO, so we cannot download the image',
      uploadbox: 'Drop file here or click to select',
      inputfile: 'Please enter the image File',
      NoISO: 'No ISO',
      sha256: 'SHA-256 (optional)',
      sha256Placeholder: 'Enter a 64-character SHA-256 checksum',
      invalidSHA256: 'SHA-256 must be a 64-character hexadecimal string',
      failed: 'Download failed',
      success: 'Download successful',
      checksumFailed: 'Download failed: SHA-256 verification failed',
      cancel: 'Cancel',
      cancelFailed: 'Failed to cancel download'
    },
    power: {
      title: 'Power',
      showConfirm: 'Confirmation',
      showConfirmTip: 'Power operations require an extra confirmation',
      reset: 'Reset',
      power: 'Power',
      powerShort: 'Power (short click)',
      powerLong: 'Power (long click)',
      resetConfirm: 'Proceed reset operation?',
      powerConfirm: 'Proceed power operation?',
      okBtn: 'Yes',
      cancelBtn: 'No'
    },
    devices: {
      title: 'Devices',
      stale: 'Live device state is unavailable. Reconnecting.',
      empty: 'No camera or microphone slots are configured. Add one under Settings, Device.',
      available: 'Available',
      waiting: 'The host is waiting for a source',
      hostOpen: 'Host open',
      hostIdle: 'Host idle',
      sending: 'Sending from this browser',
      black: 'Black video',
      silence: 'Digital silence',
      resuming: 'Waiting to resume',
      stop: 'Stop sharing',
      disconnect: 'Disconnect',
      takeover: 'Take over',
      refused: 'In use by {{owner}} from {{source}}',
      connectedSources_one: '{{count}} connected source',
      connectedSources_other: '{{count}} connected sources',
      connectedSources: '{{count}} connected sources',
      connection: {
        connecting: 'Connecting',
        connected: 'Live',
        disconnected: 'Reconnecting'
      },
      share: {
        camera: 'Share camera',
        microphone: 'Share microphone',
        usbDevice: 'Share USB'
      },
      permission: {
        denied: "Blocked in your browser's site settings",
        prompt: 'Your browser will ask for access'
      },
      mic: {
        mute: 'Mute',
        unmute: 'Unmute'
      },
      revoked: {
        released: 'Sharing was stopped',
        lease_expired: 'The lease expired before this browser came back',
        admin_disconnect: 'An administrator disconnected every source',
        slot_removed: 'The slot was removed',
        slot_changed: 'The slot was reconfigured',
        taken_over: 'An administrator took this slot over'
      },
      usb: {
        surrendered: 'USB passthrough holds the keyboard and mouse',
        surrenderedDesc:
          'The remote host sees the imported device instead of NanoKVM’s keyboard, mouse and virtual media. They come back when the session stops.',
        unsupported: 'WebUSB needs a Chromium browser',
        insecure:
          'This page is not served over HTTPS, so the browser withholds WebUSB. Enable HTTPS in Settings, Network.',
        session: 'Passing through {{device}} ({{mode}})',
        idle: 'No passthrough session',
        mode: {
          hybrid: 'hybrid',
          exact: 'exact'
        }
      }
    },
    settings: {
      title: 'Settings',
      display: {
        title: 'Display',
        loading: 'Loading...',
        active: 'Active EDID',
        activeUnknown:
          'NanoKVM has not written an EDID since it started, so the identity the host sees is unknown.',
        appliedAt: 'Applied {{time}}',
        download: 'Download',
        downloadBackup: 'Download previous',
        preset: 'Monitor preset',
        presetPlaceholder: 'Select a monitor',
        upload: 'Upload',
        selected: 'Selected EDID',
        errors: 'Errors',
        warnings: 'Warnings',
        info: 'Information',
        unknownMonitor: 'Unknown monitor',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Audio',
        audioNo: 'No audio',
        extensionBlocks: 'Extension blocks: {{blocks}}',
        apply: 'Apply',
        applyTitle: 'Apply this EDID?',
        before: 'Current',
        after: 'New',
        hdmiNotice:
          'Video capture stops while the EDID is written and starts again on its own afterwards.',
        powerCycleNotice:
          'This device must be physically unplugged from power and plugged back in before the new EDID takes effect.',
        powerCycleUnverified:
          'The write did not verify, so the video chip keeps whatever it now holds until this device is physically unplugged from power and plugged back in.',
        applied: 'EDID applied and verified.',
        applyFailed: 'Applying the EDID failed.',
        busy: 'The video chip was busy. Try again.',
        unsupported: 'This device does not support changing the EDID.',
        toolMissing: 'The EDID tool is missing from this firmware.',
        noAudio: 'This EDID advertises no audio, so the host may stop sending sound.',
        oldVersion: 'This EDID uses a version older than 1.4.',
        interlaced: 'The preferred timing is interlaced.',
        tooLarge:
          'The preferred timing is larger than 1920x1080 at 60 Hz, which is more than NanoKVM can capture.',
        recovery: 'Recovery',
        recoveryNeeded:
          'The last write did not verify, so the EDID area of the video chip is in an unknown state. Restore the factory EDID to make it known again.',
        recoveryDesc:
          'Write a known EDID back to the video chip when an applied one left the host without a picture.',
        restoreFactory: 'Restore factory EDID',
        restoreBackup: 'Restore previous EDID',
        restoreTitle: 'Restore this EDID?',
        restoreFactoryTarget: 'The factory EDID that NanoKVM shipped with.',
        restoreBackupTarget: 'The most recent backup, applied {{time}}.',
        restoreNotice: 'A restore is written the same way as an apply, with the same consequences.',
        restored: 'EDID restored and verified.',
        restoreFailed: 'Restoring the EDID failed.',
        mismatchTitle: 'What was written and what came back',
        mismatchWritten: 'Written',
        mismatchRead: 'Read back',
        restoreOkBtn: 'Restore',
        hardware: 'Detected hardware: {{hardware}}',
        hardwareUnknown: 'Unknown',
        confirmWord: 'APPLY',
        confirmPrompt: 'Type {{word}} to enable the apply button.',
        okBtn: 'Apply',
        cancelBtn: 'Cancel'
      },
      presentation: {
        title: 'Presentation',
        loading: 'Loading...',
        current: 'Current USB presentation',
        noProfile: 'No profile applied',
        linked: 'Linked functions',
        hostState: 'Host USB',
        hostUnbound: 'Controller not bound',
        hdmiState: 'HDMI input',
        hdmiSignal: 'Signal present',
        hdmiUnreported: 'No capture report yet',
        endpoints: 'Endpoints',
        fifos: 'FIFO slots',
        pending: 'Pending changes',
        pendingEdits: 'Unsaved identity edits',
        pendingProfile: '{{profile}} is selected but not applied',
        pendingNone: 'None',
        lastApply: 'Last apply',
        applyFailed: 'Failed on {{profile}} at {{time}}',
        applyClean: 'No failure recorded',
        lastKnownGood: 'Last known good',
        rollbackTarget: 'Rollback target',
        rollbackNone: 'None',
        powerCyclePending:
          'The controller was taken from the host. Power-cycle the connected computer to get the device back.',
        rollback: 'Roll back',
        rollbackTitle: 'Roll back to {{profile}}?',
        rollbackDesc: 'The gadget re-enumerates; USB functions drop briefly.',
        profile: 'USB profile',
        builtIn: 'built-in',
        descriptors: 'descriptors',
        imported: 'imported',
        clone: 'Clone',
        cloneTitle: 'Clone this profile',
        cloneToEdit: 'Built-in profiles stay read-only. Clone this profile to edit its identity.',
        profileName: 'Profile name',
        profileNameHint: 'Lowercase letters, numbers, periods, underscores, and hyphens.',
        import: 'Import package',
        export: 'Export package',
        delete: 'Delete',
        deleteTitle: 'Delete this profile?',
        deleteDesc: 'Delete {{profile}} from NanoKVM.',
        identity: 'USB identity',
        preset: 'Preset identity',
        presetPlaceholder: 'Copy identity from a known device',
        presetHint:
          'A preset fills the vendor ID, the product ID and the two name fields. It carries no descriptors.',
        presetSource: 'Identity as recorded in {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'This vendor ID belongs to another manufacturer',
        productId: 'Product ID',
        bcdUSB: 'USB version',
        bcdDevice: 'Device version',
        manufacturer: 'Manufacturer',
        product: 'Product',
        serial: 'Serial number',
        configuration: 'Configuration string',
        hidLayout: 'HID devices',
        hidRoleKeyboard: 'Keyboard',
        hidRoleRelative: 'Mouse (relative)',
        hidRoleAbsolute: 'Pointer (absolute)',
        hidOff: 'Not present',
        hidInterface: 'Interface {{index}}',
        hidBootKeyboardShared:
          'The keyboard shares an interface, so it no longer offers a boot-protocol report. Some BIOS and UEFI setups will not see it.',
        functions: 'Functions',
        descriptorAssets: 'Stored descriptor assets: {{count}}',
        endpointUse: 'IN {{inUse}} used, {{inFree}} free; OUT {{outUse}} used, {{outFree}} free',
        apply: 'Apply',
        applyTitle: 'Apply this USB profile?',
        applyDesc: 'NanoKVM will present {{profile}} to the connected computer.',
        reconnect:
          'Keyboard, mouse and other USB functions disconnect briefly while the gadget is rebound.',
        applyLinks: 'Links: {{functions}}',
        applyRemoves: 'Removes: {{functions}}',
        applyNoHid: 'No HID function remains after this apply. Keyboard and mouse stop working.',
        applyRollback: 'A failed apply returns to {{profile}}.',
        recoveryPowerCycle:
          'No HID survives this apply, so a host that stops responding can only be recovered by a power cycle.',
        recoveryReboot:
          'An interface disappears from the composite device; the host may need a reboot to rebind the rest.',
        recoveryHdmiReset: 'A video function is rebuilt, so the capture pipeline behind it resets.',
        recoveryReconnect: 'The host re-enumerates the device; USB functions drop briefly.',
        cancel: 'Cancel',
        noFunctions: 'No linked functions',
        loadFailed: 'Failed to load presentation profiles',
        operationFailed: 'Presentation operation failed'
      },
      passthrough: {
        title: 'USB Passthrough',
        loading: 'Loading...',
        mode: 'Mode',
        hybrid: 'Hybrid',
        exact: 'Exact',
        hybridDesc: 'Keeps the boot keyboard and relative mouse, for compatible devices.',
        exactDesc: 'Replaces every NanoKVM USB function with the imported device.',
        hybridWarning: 'Hybrid keeps the keyboard and relative mouse available',
        hybridWarningDesc:
          'Storage, USB networking and the absolute pointer disconnect while the imported function is active.',
        hidWarning: 'Starting passthrough gives up the keyboard, the mouse and virtual media',
        hidWarningDesc:
          'NanoKVM has one USB device controller and the proxy needs all of it, so while a session runs the remote host sees the passed-through device instead of NanoKVM’s keyboard, mouse and virtual media. They come back on their own the moment the session stops. This web interface is unaffected, so you can always stop a session from this page.',
        hidWarningSafeDesc:
          'NanoKVM has one USB device controller and the proxy needs all of it, so while a session runs the remote host sees the passed-through device instead of NanoKVM’s keyboard, mouse and virtual media. They return when the session stops.',
        isoLabel: 'Allow isochronous transfers',
        isoHint:
          'Lets webcams, microphones and other streaming devices through. Nobody has measured what this hardware sustains.',
        isoWarning:
          'Isochronous streaming is unproven here and can hold the keyboard and mouse until you stop the session',
        info: {
          title: 'Info',
          hybrid:
            'Hybrid mode keeps the keyboard and the relative mouse available. Storage, USB networking and the absolute pointer disconnect while the imported device is active.',
          exact:
            'Exact mode replaces every NanoKVM USB function with the imported device. The keyboard, the mouse and virtual media come back on their own when the session stops.',
          udc: 'NanoKVM has one USB device controller and the proxy needs all of it, which is why the functions above go away for as long as a session runs.',
          web: 'This web interface is unaffected, so you can always stop a session from this page.',
          network:
            'Start passthrough over Ethernet or Wi-Fi. Starting it from NanoKVM’s USB network is refused, because that connection would disappear.',
          iso: 'Webcams, microphones and other isochronous devices are refused unless you allow isochronous transfers. That path works but has never been measured on this hardware, so treat its throughput as unknown.',
          camera:
            'The browser camera and microphone under Devices remain the proven way to give the target one.'
        },
        session: 'Session',
        activeDesc: 'A device is imported and the proxy is holding the USB controller.',
        inactiveDesc:
          'No session is running. The keyboard, the mouse and virtual media are working normally.',
        device: 'Device',
        busId: 'Bus ID',
        speed: 'Speed',
        exporter: 'Exporter',
        local: 'Imported as',
        localValue: 'Bus {{bus}}, address {{address}}',
        udc: 'USB controller',
        pid: 'Proxy PID',
        startedAt: 'Started',
        isoDevice:
          'This device streams over isochronous endpoints, which has never been measured on this hardware',
        exporterLabel: 'Exporter address',
        exporterHint:
          'The host and port NanoKVM dials. Over the tunnel below that is {{exporter}}.',
        busIdLabel: 'Bus ID on your machine',
        busIdHint: 'The busid that usbip list -l prints for the device, for example {{example}}.',
        start: 'Start passthrough',
        stop: 'Stop passthrough',
        startTitle: 'Start USB passthrough?',
        startDevice: 'NanoKVM will import {{busId}} from {{exporter}}.',
        startHid:
          'The USB keyboard, the mouse and virtual media stop working for as long as the session runs, and start again on their own when you stop it.',
        startIso:
          'Webcams and other isochronous devices need the isochronous switch turned on before you start.',
        startWeb:
          'This web interface keeps working, so you can stop the session from this page at any time.',
        startNetwork:
          'Use this page over Ethernet or Wi-Fi. Starting from NanoKVM’s USB network is refused because that connection would disappear.',
        okBtn: 'Start',
        cancelBtn: 'Cancel',
        instructions: 'On your own machine',
        instructionsDesc:
          'There is no client agent to install, by design. Run these stock usbip commands on the machine that owns the device.',
        copyFailed: 'Copy failed. Copy the command manually.',
        copyInsecure:
          'This page is not served over HTTPS, so the browser blocks copying. Copy the command manually, or enable HTTPS in Settings, Network.',
        directNote:
          'Without a tunnel, usbipd has to be reachable on your network and the exporter address above has to name it. usbip carries the device unencrypted, so prefer the tunnel.',
        steps: {
          modprobe: {
            title: 'Load the exporter driver',
            desc: 'usbip-host is what lets your kernel hand a local device over. It is not loaded by default.'
          },
          list: {
            title: 'Find the device',
            desc: 'Prints every local device with its busid and its vendor:product pair. Note the busid of the one you want.'
          },
          bind: {
            title: 'Bind it to usbip',
            desc: 'Takes the device away from its normal driver, so it stops working on this machine until you unbind it.'
          },
          serve: {
            title: 'Serve it',
            desc: 'usbipd stays in the foreground and waits for NanoKVM to import the device.',
            notice:
              'Stock usbipd has no listen address option and binds every interface. Keep port {{port}} closed at your firewall and let only the tunnel below reach it.'
          },
          tunnel: {
            title: 'Point it at NanoKVM',
            desc: 'An SSH reverse tunnel: port {{port}} on NanoKVM’s own loopback becomes usbipd on this machine. Leave it running for the whole session.'
          },
          exporter: {
            title: 'Use this as the exporter',
            desc: 'Put this in the exporter field above, enter the bus ID, then start the session.'
          },
          unbind: {
            title: 'Give the device back',
            desc: 'After the session stops, this returns the device to its normal driver on this machine.'
          }
        }
      },
      mcp: {
        title: 'MCP Service',
        service: 'Remote control MCP',
        serviceDesc:
          'Allow trusted MCP clients to control the keyboard and mouse and capture screenshots',
        securityWarning:
          'Anyone with this API key can control the remote host and view its screen. Use HTTPS and enable it only on trusted networks.',
        endpoint: 'Endpoint',
        apiKey: 'API Key',
        regenerateConfirmTitle: 'Regenerate MCP API key?',
        regenerateConfirmDesc: 'The current key will stop working immediately.',
        enableConfirmTitle: 'Enable external MCP control?',
        enableConfirmDesc: 'Enabling MCP will stop PicoClaw and close any active PicoClaw session.',
        failed: 'MCP operation failed',
        copyFailed: 'Copy failed. Copy manually.',
        copyInsecure:
          'This page is not served over HTTPS, so the browser blocks copying. Copy manually, or enable HTTPS in Settings, Network.',
        okBtn: 'Confirm',
        cancelBtn: 'Cancel'
      },
      about: {
        title: 'About NanoKVM',
        information: 'Information',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Application Version',
        applicationTip: 'NanoKVM web application version',
        image: 'Image Version',
        imageTip: 'NanoKVM system image version',
        deviceKey: 'Device Key',
        community: 'Community',
        hostname: 'Hostname',
        hostnameUpdated: 'Hostname updated. Reboot to apply.',
        ipType: {
          Wired: 'Wired',
          Wireless: 'Wireless',
          Other: 'Other'
        }
      },
      appearance: {
        title: 'Appearance',
        display: 'Display',
        language: 'Language',
        languageDesc: 'Select the language for the interface',
        webTitle: 'Web Title',
        webTitleDesc: 'Customize the web page title',
        favicon: 'Favicon',
        faviconDesc: 'Customize the browser tab icon',
        faviconPlaceholder: 'Image URL',
        faviconUpload: 'Upload',
        faviconReset: 'Reset',
        faviconCustom: 'Custom icon',
        faviconBoot: 'Icon from /boot/logo.ico',
        faviconDefault: 'Default icon',
        faviconOverridesBoot: 'Overriding /boot/logo.ico',
        faviconErrUrl: 'Enter an http:// or https:// image address',
        faviconErrFetch: 'The device could not download that image',
        faviconErrLarge: 'Image is too large. The limit is 256 KB',
        faviconErrType: 'Unsupported image. Use .ico, .png, .svg, .gif or .jpg',
        faviconErrSave: 'Failed to save the icon',
        menuBar: {
          title: 'Menu Bar',
          mode: 'Display Mode',
          modeDesc: 'Display menu bar on the screen',
          modeOff: 'Off',
          modeAuto: 'Auto hide',
          modeAlways: 'Always visible',
          keyboardLedStatus: 'Keyboard lock indicators',
          keyboardLedStatusDesc: 'Display remote Num Lock, Caps Lock, and Scroll Lock status',
          icons: 'Submenu Icons',
          iconsDesc: 'Display submenu icons in the menu bar'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Remote keyboard lock status',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'On',
        off: 'Off',
        unknown: 'Unknown'
      },
      device: {
        title: 'Device',
        oled: {
          title: 'OLED',
          description: 'Turn off OLED screen after',
          0: 'Never',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 hour'
        },
        ssh: {
          description: 'Enable SSH remote access',
          tip: 'Set a strong password before enabling (Account - Change Password)'
        },
        advanced: 'Advanced Settings',
        swap: {
          title: 'Swap',
          disable: 'Disable',
          description: 'Set the swap file size',
          tip: "Enabling this feature could shorten your SD card's usable life!"
        },
        mouseJiggler: {
          title: 'Mouse Jiggler',
          description: 'Prevent the remote host from sleeping',
          disable: 'Disable',
          absolute: 'Absolute Mode',
          relative: 'Relative Mode'
        },
        mdns: {
          description: 'Enable mDNS discovery service',
          tip: "Turning it off if it's not needed"
        },
        hdmi: {
          description: 'Enable HDMI/monitor output',
          idleTimeoutTitle: 'Capture idle timeout',
          idleTimeoutDescription: 'Stop HDMI capture after there are no active viewers for',
          minutes: 'min'
        },
        autostart: {
          title: 'Autostart Scripts Settings',
          description: 'Manage scripts that run automatically on system startup',
          new: 'New',
          deleteConfirm: 'Are you sure you want to delete this file?',
          yes: 'Yes',
          no: 'No',
          scriptName: 'Autostart Script Name',
          scriptContent: 'Autostart Script Content',
          settings: 'Settings'
        },
        hidOnly: 'HID-Only Mode',
        hidOnlyDesc: 'Stop emulating virtual devices, retaining only basic HID control',
        disk: 'Virtual Disk',
        diskDesc: 'Mount SD card on the remote host',
        rebindNotice:
          'Changing this switch re-enumerates the USB device, so the target briefly loses its virtual devices and its USB network.',
        media: {
          title: 'Camera and microphone slots',
          desc: 'Declare the media devices browsers may fill. The endpoint budget is checked when the USB profile is applied. Saving re-enumerates the gadget and disconnects any connected browser.',
          cameras: 'Cameras',
          microphones: 'Microphones',
          name: 'Name',
          namePlaceholder: 'Shown on the target host',
          addCamera: 'Add camera',
          addMicrophone: 'Add microphone',
          remove: 'Remove',
          cameraDefault: 'NanoKVM Camera {{index}}',
          microphoneDefault: 'NanoKVM Microphone {{index}}',
          nameRequired: 'Every slot needs a name.',
          budgetHint:
            'The six USB IN endpoints are a fixed hardware limit. Put the keyboard, mouse and pointer on one HID interface under Presentation, or turn off the virtual disk here or the USB network adapter under Network.',
          unsupported: 'This kernel cannot name media devices, so hosts show the default name.',
          save: 'Save slots',
          disconnect: 'Disconnect',
          disconnectAll: 'Disconnect all sources',
          limit: 'Camera and microphone slots must total eight or fewer.',
          failed: 'Media slots could not be updated.'
        },
        reboot: 'Reboot',
        rebootDesc: 'Are you sure you want to reboot NanoKVM?',
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      network: {
        title: 'Network',
        wifi: {
          title: 'Wi-Fi',
          description: 'Configure Wi-Fi',
          apMode: 'AP mode is enabled, connect to Wi-Fi by scanning QR code',
          connect: 'Join Wi-Fi',
          connectDesc1: 'Please enter the network ssid and password',
          connectDesc2: 'Please enter the password to join this network',
          disconnect: 'Are you sure to disconnect the network?',
          failed: 'Connection failed, please try again.',
          ssid: 'Name',
          password: 'Password',
          joinBtn: 'Join',
          confirmBtn: 'Ok',
          cancelBtn: 'Cancel'
        },
        tls: {
          description: 'Enable HTTPS protocol',
          tip: 'Be aware: Using HTTPS can increase latency, especially with MJPEG video mode.'
        },
        usb: {
          title: 'USB Network Adapter',
          desc: 'Gives the controlled computer a network card over USB',
          type: 'Adapter type',
          typeDesc: 'NCM for modern systems, RNDIS for older Windows'
        },
        bridge: {
          title: 'Adapter connects to',
          lan: 'Your network',
          kvmOnly: 'NanoKVM only',
          lanDesc:
            "The computer joins your network through NanoKVM's Ethernet port, with its own address from your router.",
          kvmOnlyDesc:
            'The computer gets its address from NanoKVM and can reach NanoKVM, but nothing beyond it.',
          loading: 'Loading...',
          state: 'State',
          states: {
            disabled: 'NanoKVM only',
            enabled: 'Your network',
            rolledBack: 'Rolled back',
            failed: 'Failed',
            pending: 'In progress'
          },
          uplink: 'Uplink',
          ports: 'Ports',
          up: 'up',
          down: 'down',
          noLink: 'no link',
          enableTitle: 'Connect the computer to your network?',
          disableTitle: 'Limit the computer to NanoKVM only?',
          reconnect: 'Management will briefly disconnect and reconnect while the address moves.',
          rollback: 'If verification fails, the previous configuration is restored automatically.',
          enableBtn: 'Join my network',
          disableBtn: 'NanoKVM only',
          cancelBtn: 'Cancel',
          interrupted:
            'The connection was interrupted while applying. Re-checking the current state.',
          pendingNotice:
            'A bridge change is still in progress, or was interrupted before it finished.',
          revert: 'Restore previous configuration',
          rolledBackNotice:
            'The last change was rolled back and the previous configuration was restored.',
          verifyFailed: 'Verification failed: {{gates}}',
          gates: {
            address: 'address',
            gateway: 'gateway',
            inbound: 'inbound connection'
          },
          inboundWeak:
            'The inbound check passed only because NanoKVM connected to itself. That proves the web service is listening and reachable on this device, not that a request from the network reaches it.',
          noCarrier:
            'No link on {{port}}. The bridge has no path to the network until a cable is connected.',
          loop: 'The router is also being learned on {{port}}, so that port is a second path to the same network. Spanning tree is off, so nothing here will break the loop: disconnect one of the two paths.',
          failedNotice:
            'The last change could not be undone. NanoKVM may only be reachable over the Wi-Fi AP or a serial console.'
        },
        dns: {
          title: 'DNS',
          description: 'Configure DNS servers for NanoKVM',
          mode: 'Mode',
          dhcp: 'DHCP',
          manual: 'Manual',
          add: 'Add DNS',
          save: 'Save',
          invalid: 'Please enter a valid IP address',
          noDhcp: 'No DHCP DNS is currently available',
          saved: 'DNS settings saved',
          saveFailed: 'Failed to save DNS settings',
          unsaved: 'Unsaved changes',
          maxServers: 'Maximum {{count}} DNS servers allowed',
          dnsServers: 'DNS Servers',
          dhcpServersDescription: 'DNS servers are automatically obtained from DHCP',
          manualServersDescription: 'DNS servers can be edited manually',
          networkDetails: 'Network Details',
          interface: 'Interface',
          ipAddress: 'IP Address',
          subnetMask: 'Subnet Mask',
          router: 'Router',
          none: 'None'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC Server',
        description:
          'Let any VNC client view the remote screen and use the keyboard and mouse, using your NanoKVM account to log in',
        port: 'Port',
        portDescription: 'Connect to this port on the NanoKVM address'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Memory optimization',
          tip: 'When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. A Tailscale restart is required for the change to take effect.'
        },
        swap: {
          title: 'Swap memory',
          tip: 'If issues persist after enabling memory optimization, try enabling swap memory. This sets the swap file size to 256MB by default, which can be adjusted in "Settings > Device".'
        },
        restart: 'Restart Tailscale?',
        stop: 'Stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable automatic startup on boot.',
        loading: 'Loading...',
        notInstall: 'Tailscale not found! Please install.',
        install: 'Install',
        installing: 'Installing',
        failed: 'Install failed',
        retry: 'Please refresh and try again. Or try to install manually',
        download: 'Download the',
        package: 'installation package',
        unzip: 'and unzip it',
        upTailscale: 'Upload tailscale to NanoKVM directory /usr/bin/',
        upTailscaled: 'Upload tailscaled to NanoKVM directory /usr/sbin/',
        refresh: 'Refresh current page',
        notRunning: 'Tailscale is not running. Please start it to continue.',
        run: 'Start',
        notLogin:
          'The device has not been bound yet. Please login and bind this device to your account.',
        urlPeriod: 'This url is valid for 10 minutes',
        login: 'Login',
        loginSuccess: 'Login Success',
        enable: 'Enable Tailscale',
        deviceName: 'Device Name',
        deviceIP: 'Device IP',
        account: 'Account',
        logout: 'Logout',
        logoutDesc: 'Are you sure you want to logout?',
        uninstall: 'Uninstall Tailscale',
        uninstallDesc: 'Are you sure you want to uninstall Tailscale?',
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Loading...',
        notInstall: 'Not installed',
        notConfigured: 'Not configured',
        stopped: 'Stopped',
        running: 'Running',
        connected: 'Connected',
        error: 'Error',
        atBoot: 'starts at boot',
        notAtBoot: 'does not start at boot',
        arguments: 'Arguments',
        argumentsTip: 'Command line arguments passed to the service at startup.',
        env: 'Environment variables',
        envKey: 'Name',
        envValue: 'Value',
        envAdd: 'Add variable',
        envRemove: 'Remove',
        configured: 'Configured',
        save: 'Save',
        saved: 'Configuration saved',
        start: 'Start',
        stop: 'Stop',
        restart: 'Restart',
        logs: 'Logs',
        logsEmpty: 'No logs yet',
        refresh: 'Refresh',
        binary: 'Binary',
        binaryShipped: 'Shipped with firmware',
        binaryCustom: 'Custom upload',
        binaryUpload: 'Upload binary',
        binaryRevert: 'Restore bundled binary',
        binaryRevertDesc:
          'Delete the uploaded binary and restore the version shipped with the firmware?',
        serverWarning: 'An unrestricted server acts as an open proxy',
        noHealthSignal:
          'This service reports no health signal, so NanoKVM can only tell that the process is running, not whether the tunnel is connected.',
        memoryWarning: 'Running several remote access services at once may exhaust memory',
        resources: 'Resources',
        memory: {
          title: 'Memory limit',
          description:
            "Caps newt's Go heap at {{limit}} MiB from its next restart. Its own limit, not Tailscale's; off leaves the Go default, with GOGC=50 applied either way.",
          noRuntime:
            'wstunnel is Rust: no garbage collector and no heap limit to set, and its worker threads already follow the device’s single CPU.',
          notApplicable: 'Not applicable'
        },
        swap: {
          title: 'Swap file',
          description:
            'Adds a 256 MB swap file on the SD card. System-wide: the same swap serves Tailscale, the KVM server and everything else on the device.'
        },
        okBtn: 'Yes',
        cancelBtn: 'No'
      },
      update: {
        title: 'Check for Updates',
        queryFailed: 'Get version failed',
        updateFailed: 'Update failed. Please retry.',
        isLatest: 'You already have the latest version.',
        rebooting:
          'Installing the new kernel and rebooting. This can take a few minutes; do not power off.',
        kernelUpdate:
          'This update installs kernel {{version}}. The device reboots, and returns to the current kernel by itself if the new one does not come back.',
        rolledBack:
          'Kernel {{version}} did not start and the device rolled back to the previous kernel.',
        available: 'An update is available. Are you sure you want to update now?',
        updating: 'Update started. Please wait...',
        confirm: 'Confirm',
        cancel: 'Cancel',
        preview: 'Preview Updates',
        previewDesc: 'Get early access to new features and improvements',
        previewTip:
          'Please be aware that preview releases may contain bugs or incomplete functionality!',
        customServer: {
          title: 'Custom Update Server',
          desc: 'Check for and download online updates from a specified server',
          invalidUrl:
            'Enter a valid HTTP or HTTPS server directory without a query, fragment, or latest.json.',
          loadFailed: 'Failed to load the update server configuration.',
          saveFailed: 'Failed to save the update server configuration.',
          saved: 'Update server configuration saved.',
          save: 'Save',
          confirmTitle: 'Use a custom update server?',
          confirmDesc:
            'SHA-512 only checks that the package matches the manifest supplied by this server. It does not prove that the package is an official NanoKVM release. A faulty or malicious server may make the device unusable, cause data loss, or compromise the system.',
          confirm: 'Use Anyway',
          previewDisabled: 'Preview Updates are unavailable while a custom update server is enabled'
        },
        offline: {
          kernelNotice:
            'This package contains a kernel. It is written to the spare slot and the device reboots to try it; if it does not come back, the device returns to the current kernel on its own.',
          kernelConfirm: 'Install Kernel',
          kernelCancel: 'Cancel',
          title: 'Offline Updates',
          desc: 'Update through local installation package',
          upload: 'Upload',
          checksumPlaceholder: 'SHA-256 checksum (optional)',
          invalidChecksum: 'The SHA-256 checksum must contain 64 hexadecimal characters.',
          checksumMismatch: 'SHA-256 verification failed. The package may be corrupted.',
          invalidName: 'Invalid filename format. Please download from GitHub releases.',
          updateFailed: 'Update failed. Please retry.'
        }
      },
      account: {
        title: 'Account',
        webAccount: 'Web Account Name',
        role: 'Role',
        roles: {
          admin: 'Administrator',
          user: 'User'
        },
        password: 'Password',
        updateBtn: 'Change',
        logoutBtn: 'Logout',
        logoutDesc: 'Are you sure you want to logout?',
        okBtn: 'Yes',
        cancelBtn: 'No',
        users: {
          title: 'Users',
          create: 'Create User',
          enabled: 'Enabled',
          disabled: 'Disabled',
          deviceOwner: 'Device owner',
          resetPassword: 'Reset Password',
          delete: 'Delete',
          deleteConfirm: 'Delete this user and revoke all of their sessions?',
          created: 'User created',
          deleted: 'User deleted',
          passwordUpdated: 'Password updated',
          loadFailed: 'Failed to load users',
          saveFailed: 'Failed to save user',
          deleteFailed: 'Failed to delete user'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistant',
      empty: 'Open the panel and start a task to begin.',
      inputPlaceholder: 'Describe what you want the PicoClaw to do',
      newConversation: 'New conversation',
      processing: 'Processing...',
      agent: {
        defaultTitle: 'General Assistant',
        defaultDescription: 'General chat, search, and workspace help.',
        kvmTitle: 'Remote Control',
        kvmDescription: 'Operate the remote host through NanoKVM.',
        switched: 'Agent role switched',
        switchFailed: 'Failed to switch agent role'
      },
      send: 'Send',
      cancel: 'Cancel',
      status: {
        connecting: 'Connecting to gateway...',
        connected: 'PicoClaw session connected',
        disconnected: 'PicoClaw session closed',
        stopped: 'Stop request sent',
        runtimeStarted: 'PicoClaw runtime started',
        runtimeStartFailed: 'Failed to start PicoClaw runtime',
        runtimeStopped: 'PicoClaw runtime stopped',
        runtimeStopFailed: 'Failed to stop PicoClaw runtime',
        controlSwitchedToMCP: 'Control switched to the external MCP service'
      },
      connection: {
        runtime: {
          checking: 'Checking',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime ready',
          stopped: 'Runtime stopped',
          blockedByMCP: 'External MCP control is active',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime unavailable',
          configError: 'Configuration error'
        },
        transport: {
          connecting: 'Connecting',
          connected: 'Connected',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Idle',
          busy: 'Busy'
        }
      },
      message: {
        toolAction: 'Action',
        observation: 'Observation',
        screenshot: 'Screenshot'
      },
      overlay: {
        locked: 'PicoClaw is controlling the device. Manual input is paused.'
      },
      control: {
        picoclaw: 'Device control: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Device control: external MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Device control: manual/no AI',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Take over',
        release: 'Return control',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw control granted',
        released: 'Device control returned',
        grantFailed: 'Failed to grant PicoClaw control',
        releaseFailed: 'Failed to release PicoClaw control',
        grantConfirmTitle: 'Switch device control to PicoClaw?',
        grantConfirmDesc: 'External MCP device writes will be interrupted.'
      },
      install: {
        install: 'Install PicoClaw',
        installing: 'Installing PicoClaw',
        success: 'PicoClaw installed successfully',
        failed: 'Failed to install PicoClaw',
        uninstalling: 'Uninstalling runtime...',
        uninstalled: 'Runtime uninstalled successfully.',
        uninstallFailed: 'Uninstall failed.',
        requiredTitle: 'PicoClaw is not installed',
        requiredDescription: 'Install PicoClaw before starting the PicoClaw runtime.',
        progressDescription: 'PicoClaw is being downloaded and installed.',
        stages: {
          preparing: 'Preparing',
          downloading: 'Downloading',
          extracting: 'Extracting',
          verifying: 'Verifying',
          installing: 'Installing',
          installed: 'Installed',
          install_timeout: 'Timed Out',
          install_failed: 'Failed'
        }
      },
      model: {
        requiredTitle: 'Model configuration is required',
        requiredDescription: 'Configure the PicoClaw model before using PicoClaw chat.',
        docsTitle: 'Configuration Guide',
        docsDesc: 'Supported models and protocols',
        menuLabel: 'Configure model',
        modelIdentifier: 'Model Identifier',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API Key',
        apiKeyPlaceholder: 'Enter the model API key',
        save: 'Save',
        saving: 'Saving',
        saved: 'Model configuration saved',
        saveFailed: 'Failed to save model configuration',
        invalid: 'Model identifier, API base URL, and API key are required'
      },
      uninstall: {
        menuLabel: 'Uninstall',
        confirmTitle: 'Uninstall PicoClaw',
        confirmContent:
          'Are you sure you want to uninstall PicoClaw? This will delete the executable and all configuration files.',
        confirmOk: 'Uninstall',
        confirmCancel: 'Cancel'
      },
      history: {
        title: 'History',
        loading: 'Loading sessions...',
        emptyTitle: 'No history yet',
        emptyDescription: 'Previous PicoClaw sessions will appear here.',
        loadFailed: 'Failed to load session history',
        deleteFailed: 'Failed to delete session',
        deleteConfirmTitle: 'Delete session',
        deleteConfirmContent: 'Are you sure you want to delete "{{title}}"?',
        deleteConfirmOk: 'Delete',
        deleteConfirmCancel: 'Cancel',
        messageCount_one: '{{count}} message',
        messageCount_other: '{{count}} messages',
        messageCount: '{{count}} messages'
      },
      config: {
        startRuntime: 'Start PicoClaw',
        stopRuntime: 'Stop PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Switch control to PicoClaw?',
        enableConfirmDesc: 'External MCP device writes will be interrupted before PicoClaw starts.',
        enableConfirmOk: 'Start PicoClaw',
        enableConfirmCancel: 'Cancel',
        title: 'Start PicoClaw',
        description: 'Start the runtime to begin using the PicoClaw assistant.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: "We've ran into an issue",
      refresh: 'Refresh'
    },
    fullscreen: {
      toggle: 'Toggle Fullscreen'
    },
    menu: {
      collapse: 'Collapse Menu',
      expand: 'Expand Menu'
    }
  }
};

export default en;
