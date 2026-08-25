const de = {
  translation: {
    head: {
      desktop: 'Entfernter Desktop',
      login: 'Anmelden',
      changePassword: 'Passwort ändern',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Anmelden',
      placeholderUsername: 'Benutzername',
      placeholderPassword: 'Passwort',
      placeholderCurrentPassword: 'Aktuelles Passwort',
      placeholderPassword2: 'Bitte Passwort erneut eingeben',
      noEmptyUsername: 'Benutzername benötigt',
      noEmptyPassword: 'Passwort benötigt',
      passwordLength: 'Das Passwort muss zwischen 8 und 72 Zeichen lang sein',
      noAccount:
        'Abfragen der Benutzerdaten fehlgeschlagen, bitte die Seite neu laden oder Passwort zurücksetzen',
      invalidUser: 'Falscher Benutzername oder falsches Passwort',
      locked: 'Zu viele Anmeldungen, bitte versuchen Sie es später noch einmal',
      globalLocked: 'System wird geschützt, bitte versuchen Sie es später erneut',
      error: 'Unerwarteter Fehler',
      invalidCurrentPassword: 'Das aktuelle Passwort ist falsch',
      changePassword: 'Passwort ändern',
      changePasswordDesc: 'Für die Sicherheit Ihres Geräts ändern Sie bitte das Passwort!',
      differentPassword: 'Passwörter stimmen nicht überein',
      illegalUsername: 'Benutzername enthält ungültige Zeichen',
      illegalPassword: 'Passwort enthält ungültige Zeichen',
      forgetPassword: 'Passwort vergessen',
      ok: 'Ok',
      cancel: 'Abbrechen',
      loginButtonText: 'Anmelden',
      tips: {
        reset1:
          'Um das Passwort zurückzusetzen, drücken und halten Sie den BOOT Knopf auf dem NanoKVM für 10 Sekunden.',
        reset2: 'Für detailliertere Anweisungen lesen Sie folgendes Dokument:',
        reset3: 'Web Standard-Account:',
        reset4: 'SSH Standard-Account:',
        change1: 'Bitte beachten Sie, dass diese Aktion folgende Passwörter ändert:',
        change2: 'Web Anmelde-Passwort',
        change3: 'System root Passwort (SSH Anmelde-Passwort)',
        change4:
          'Um die Passwörter zurückzusetzen, drücken und halten Sie den BOOT Knopf auf dem NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Wi-Fi Konfiguration für NanoKVM',
      success:
        'Bitte überprüfen Sie den Netzwerk-Status des NanoKVM und greifen Sie über die neue IP Adresse darauf zu.',
      failed: 'Aktion fehlgeschlagen, bitte erneut versuchen.',
      invalidMode:
        'Der aktuelle Modus unterstützt keine Netzwerkeinrichtung. Bitte gehen Sie zu Ihrem Gerät und aktivieren Sie den Wi-Fi-Konfigurationsmodus.',
      confirmBtn: 'Ok',
      finishBtn: 'Fertig',
      ap: {
        authTitle: 'Authentifizierung erforderlich',
        authDescription: 'Bitte geben Sie das Passwort AP ein, um fortzufahren',
        authFailed: 'Ungültiges AP Passwort',
        passPlaceholder: 'AP Passwort',
        verifyBtn: 'Überprüfen'
      }
    },
    screen: {
      scale: 'Skala',
      title: 'Bildschirm',
      video: 'Video Modus',
      videoDirectTips:
        'Aktivieren Sie HTTPS unter „Einstellungen > Gerät“, um diesen Modus zu verwenden',
      resolution: 'Auflösung',
      auto: 'Automatisch',
      autoTips:
        'Bildverzerrungen oder ein versetzter Mauszeiger können bei bestimmten Auflösungen auftreten. Versuchen Sie, die Auflösung des entfernten Hosts anzupassen oder den automatischen Modus zu deaktivieren.',
      fps: 'FPS',
      customizeFps: 'Anpassen',
      quality: 'Qualität',
      qualityLossless: 'Verlustfrei',
      qualityHigh: 'Hoch',
      qualityMedium: 'Mittel',
      qualityLow: 'Niedrig',
      frameDetect: 'Bilderkennung',
      frameDetectTip:
        'Berechnet den Unterschied zwischen den Einzelbildern. Beendet die Liveübertragung des Videostreams wenn keine Änderungen auf dem Bildschirm des Hosts festgestellt werden kann.',
      resetHdmi: 'HDMI zurücksetzen',
      mixedH264: {
        title: 'H.264-Streamkonflikt',
        description:
          'H.264 Direct und H.264 WebRTC werden gleichzeitig verwendet. Dies kann zu Bildschirm-Tearing oder beschädigtem Video führen. Bitte verwenden Sie nur einen H.264-Modus.'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC-Verbindung fehlgeschlagen',
        description: 'Überprüfen Sie die Netzwerkverbindung oder wechseln Sie den Videomodus.'
      },
      captureStatus: {
        hdmiError: 'HDMI-Bildschirmfehler',
        unsupportedResolution: 'Die aktuelle Auflösung wird nicht unterstützt',
        retrieving: 'Bildschirm wird abgerufen...',
        changingResolution: 'Auflösung wird gewechselt...',
        updateFailed: 'Der Bildschirm kann derzeit nicht aktualisiert werden',
        videoError: 'Fehler bei der Videoanzeige',
        noHdmi: 'Kein HDMI-Signal erkannt',
        unavailable: 'Der Bildschirm kann derzeit nicht angezeigt werden'
      }
    },
    keyboard: {
      title: 'Tastatur',
      paste: 'Einfügen',
      tips: 'Server Tastaturbelegung',
      placeholder: 'Bitte eingeben',
      submit: 'Senden',
      virtual: 'Tastatur',
      readClipboard: 'Aus der Zwischenablage lesen',
      clipboardPermissionDenied:
        'Berechtigung für die Zwischenablage verweigert. Bitte erlauben Sie den Zugriff auf die Zwischenablage in Ihrem Browser.',
      clipboardReadError: 'Zwischenablage konnte nicht gelesen werden',
      dropdownEnglish: 'Englisch',
      dropdownGerman: 'Deutsch',
      dropdownFrench: 'Französisch',
      dropdownRussian: 'Russisch',
      shortcut: {
        title: 'Verknüpfungen',
        custom: 'Benutzerdefiniert',
        capture: 'Klicken Sie hier, um die Verknüpfung zu erfassen',
        clear: 'Klar',
        save: 'Speichern',
        captureTips:
          'Das Erfassen systemweiter Tasten (z. B. der Windows-Taste) erfordert die Vollbildberechtigung.',
        enterFullScreen: 'Vollbildmodus umschalten.'
      },
      leaderKey: {
        title: 'Leader-Taste',
        desc: 'Browserbeschränkungen umgehen und Systemverknüpfungen direkt an den Remote-Host senden.',
        howToUse: 'Verwendung',
        simultaneous: {
          title: 'Simultanmodus',
          desc1: 'Halten Sie die Leader-Taste gedrückt und drücken Sie dann die Tastenkombination.',
          desc2: 'Intuitiv, kann jedoch zu Konflikten mit Systemverknüpfungen führen.'
        },
        sequential: {
          title: 'Sequenzieller Modus',
          desc1:
            'Drücken Sie die Leader-Taste → drücken Sie die Tastenkombination nacheinander → drücken Sie erneut die Leader-Taste.',
          desc2: 'Erfordert mehr Schritte, vermeidet jedoch vollständig Systemkonflikte.'
        },
        enable: 'Leader-Taste aktivieren',
        tip: 'Wenn diese Taste als Leader-Taste zugewiesen wird, dient sie ausschließlich als Auslöser für Tastenkombinationen und verliert ihr Standardverhalten.',
        placeholder: 'Bitte drücken Sie die Leader-Taste',
        shiftRight: 'Rechts Shift',
        ctrlRight: 'Rechts Ctrl',
        metaRight: 'Rechts Win',
        submit: 'Senden',
        recorder: {
          rec: 'REC',
          activate: 'Tasten aktivieren',
          input: 'Bitte drücken Sie die Tastenkombination...'
        }
      }
    },
    mouse: {
      title: 'Maus',
      cursor: 'Cursor',
      default: 'Standard Cursor',
      pointer: 'Zeiger Cursor',
      cell: 'Zellen Cursor',
      text: 'Text Cursor',
      grab: 'Greif Cursor',
      hide: 'Versteckter Cursor',
      mode: 'Maus Modus',
      absolute: 'Absoluter Modus',
      relative: 'Relativer Modus',
      direction: 'Scrollrichtung',
      scrollUp: 'Nach oben scrollen',
      scrollDown: 'Scrollen Sie nach unten',
      speed: 'Scrollgeschwindigkeit',
      fast: 'Schnell',
      slow: 'Langsam',
      requestPointer:
        'Relativer Modus aktiv. Klicken Sie auf den Desktop um den Mauszeiger zu sehen.',
      resetHid: 'HID zurücksetzen',
      hidOnly: {
        title: 'HID-Only-Modus',
        desc: 'Wenn Ihre Maus und Tastatur nicht mehr reagieren und das Zurücksetzen der HID-Verbindung nicht hilft, könnte es sich um ein Kompatibilitätsproblem zwischen dem NanoKVM und dem Gerät handeln. Versuchen Sie, den HID-Only Modus zu aktivieren, um die Kompatibilität zu verbessern.',
        tip1: 'Die Aktivierung des HID-Only Modus entfernt das virtuelle U-Laufwerk und das virtuelle Netzwerk.',
        tip2: 'Im HID-Only Modus ist das Einbinden von Systemabbilder deaktiviert.',
        tip3: 'NanoKVM wird nach dem Wechsel in den neuen Modus automatisch neu gestartet.',
        enable: 'HID-Only Modus aktivieren',
        disable: 'HID-Only Modus deaktivieren'
      }
    },
    image: {
      title: 'Bilder',
      loading: 'Lädt...',
      empty: 'Nichts gefunden',
      mountMode: 'Mount-Modus',
      mountFailed: 'Einbinden fehlgeschlagen',
      mountDesc:
        'In einigen Systemen ist es notwendig, die virtuelle Festplatte auf dem entfernten Host auszuwerfen, bevor das Image eingebunden werden kann.',
      unmountFailed: 'Das Aufheben der Bereitstellung ist fehlgeschlagen',
      unmountDesc:
        'Auf einigen Systemen müssen Sie das Image manuell vom Remote-Host auswerfen, bevor Sie die Bereitstellung aufheben.',
      refresh: 'Bilder aktualisieren',
      attention: 'Achtung',
      deleteConfirm: 'Sind Sie sicher, dass Sie dieses Bild löschen möchten?',
      okBtn: 'Ja',
      cancelBtn: 'Nein',
      tips: {
        title: 'So laden Sie Dateien hoch',
        usb1: 'Verbinden Sie den NanoKVM über USB mit Ihrem Computer.',
        usb2: 'Stellen Sie sicher, dass die virtuelle Festplatte eingebunden ist (Einstellungen – Virtuelle Festplatte).',
        usb3: 'Öffnen Sie die virtuelle Festplatte auf Ihrem Computer und kopieren Sie die Image-Datei in das Stammverzeichnis der virtuellen Festplatte.',
        scp1: 'Stellen Sie sicher, dass sich der NanoKVM und Ihr Computer im selben lokalen Netzwerk befinden.',
        scp2: 'Öffnen Sie ein Terminal auf Ihrem Computer und verwenden Sie den SCP-Befehl, um die Image-Datei in das Verzeichnis /data auf dem NanoKVM hochzuladen.',
        scp3: 'Beispiel: scp your-image-path root@your-nanokvm-ip:/data',
        tfCard: 'TF-Karte',
        tf1: 'Diese Methode wird unter Linux-Systemen unterstützt.',
        tf2: 'Entnehmen Sie die TF-Karte aus dem NanoKVM (bei der FULL-Version muss zuvor das Gehäuse geöffnet werden).',
        tf3: 'Stecken Sie die TF-Karte in einen Kartenleser und verbinden Sie diesen mit Ihrem Computer.',
        tf4: 'Kopieren Sie die Image-Datei in das Verzeichnis /data auf der TF-Karte.',
        tf5: 'Setzen Sie die TF-Karte wieder in den NanoKVM ein.'
      }
    },
    script: {
      title: 'Skripte',
      upload: 'Hochladen',
      run: 'Ausführen',
      runBackground: 'Im Hintergrund ausführen',
      runFailed: '',
      attention: 'Achtung',
      delDesc: 'Möchten Sie diese Datei wirklich löschen?',
      confirm: 'Ja',
      cancel: 'Nein',
      delete: 'Löschen',
      close: 'Schliessen'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM Terminal',
      serial: 'Serieller Anschluss Terminal',
      serialPort: 'Serieller Anschluss',
      serialPortPlaceholder: 'Bitte seriellen Anschluss angeben',
      baudrate: 'Baudrate',
      parity: 'Parität',
      parityNone: 'Keine',
      parityEven: 'Gerade',
      parityOdd: 'Ungerade',
      flowControl: 'Fluss-Kontrolle',
      flowControlNone: 'Keine',
      flowControlSoft: 'Software',
      flowControlHard: 'Hardware',
      dataBits: 'Daten bits',
      stopBits: 'Stopp bits',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Sende Befehl...',
      sent: 'Befehl gesendet',
      input: 'Bitte MAC Adresse eingeben',
      ok: 'Ok'
    },
    download: {
      title: 'Systemabbild Downloader',
      input: 'Bitte geben Sie die URL für das Remote-Systemabbild ein',
      ok: 'Ok',
      disabled:
        '/data Partition ist nur-lesbar, daher kann das Systemabbild nicht heruntergeladen werden',
      uploadbox: 'Datei hier ablegen oder klicken zum Auswählen',
      inputfile: 'Bitte geben Sie die Datei für das Systemabbild an',
      NoISO: 'Keine ISO',
      sha256: 'SHA-256 (optional)',
      sha256Placeholder: 'Geben Sie eine 64-stellige SHA-256-Prüfsumme ein',
      invalidSHA256: 'SHA-256 muss eine 64-stellige Hexadezimalzeichenfolge sein',
      failed: 'Download fehlgeschlagen',
      success: 'Download erfolgreich',
      checksumFailed: 'Download fehlgeschlagen: SHA-256-Prüfung fehlgeschlagen',
      cancel: 'Abbrechen',
      cancelFailed: 'Download konnte nicht abgebrochen werden'
    },
    power: {
      title: 'Power',
      showConfirm: 'Bestätigung',
      showConfirmTip: 'Diese Aktionen benötigen eine zusätzliche Bestätigung',
      reset: 'Zurücksetzen',
      power: 'Power',
      powerShort: 'Power (Kurzer Klick)',
      powerLong: 'Power (Langer Klick)',
      resetConfirm: 'Reset-Aktion durchführen?',
      powerConfirm: 'Power-Aktion durchführen?',
      okBtn: 'Ja',
      cancelBtn: 'Nein'
    },
    devices: {
      title: 'Geräte',
      stale: 'Der Live-Status der Geräte ist nicht verfügbar. Verbindung wird wiederhergestellt.',
      empty:
        'Es sind keine Kamera- oder Mikrofonplätze eingerichtet. Unter Einstellungen, Gerät einen hinzufügen.',
      available: 'Verfügbar',
      waiting: 'Der Host wartet auf eine Quelle',
      hostOpen: 'Host geöffnet',
      hostIdle: 'Host untätig',
      sending: 'Sendet aus diesem Browser',
      black: 'Schwarzes Video',
      silence: 'Digitale Stille',
      resuming: 'Wartet auf Fortsetzung',
      stop: 'Freigabe beenden',
      disconnect: 'Trennen',
      takeover: 'Übernehmen',
      refused: 'In Benutzung von {{owner}} über {{source}}',
      connectedSources_one: '{{count}} verbundene Quelle',
      connectedSources_other: '{{count}} verbundene Quellen',
      connectedSources: '{{count}} verbundene Quellen',
      connection: {
        connecting: 'Verbindung wird hergestellt',
        connected: 'Live',
        disconnected: 'Verbindung wird wiederhergestellt'
      },
      share: {
        camera: 'Kamera freigeben',
        microphone: 'Mikrofon freigeben',
        usbDevice: 'USB freigeben'
      },
      permission: {
        denied: 'In den Website-Einstellungen Ihres Browsers blockiert',
        prompt: 'Ihr Browser wird nach der Freigabe fragen',
        insecure:
          'Diese Seite wird nicht über HTTPS ausgeliefert, deshalb sperrt der Browser dieses Gerät. HTTPS unter Einstellungen, Netzwerk aktivieren.'
      },
      capture: {
        unsupported: 'Dieser Browser kann kein Audio und kein Video aufnehmen',
        camera: 'Dieser Browser kann keine Kamerabilder kodieren',
        microphone: 'Dieser Browser kann kein Mikrofonaudio verarbeiten'
      },
      mic: {
        mute: 'Stummschalten',
        unmute: 'Stummschaltung aufheben'
      },
      revoked: {
        released: 'Die Freigabe wurde beendet',
        lease_expired: 'Die Lease lief ab, bevor dieser Browser zurückkam',
        admin_disconnect: 'Ein Administrator hat alle Quellen getrennt',
        slot_removed: 'Der Slot wurde entfernt',
        slot_changed: 'Der Slot wurde neu konfiguriert',
        taken_over: 'Ein Administrator hat diesen Slot übernommen'
      },
      usb: {
        surrendered: 'Das USB-Passthrough hält Tastatur und Maus',
        surrenderedDesc:
          'Der entfernte Host sieht das importierte Gerät statt Tastatur, Maus und virtuellen Medien des NanoKVM. Sie kommen zurück, sobald die Sitzung endet.',
        unsupported: 'WebUSB braucht einen Chromium-Browser',
        insecure:
          'Diese Seite wird nicht über HTTPS ausgeliefert, deshalb hält der Browser WebUSB zurück. HTTPS unter Einstellungen, Netzwerk aktivieren.',
        session: '{{device}} wird durchgereicht ({{mode}})',
        idle: 'Keine Passthrough-Sitzung',
        mode: {
          hybrid: 'hybrid',
          exact: 'exakt'
        }
      }
    },
    settings: {
      title: 'Einstellungen',
      display: {
        title: 'Anzeige',
        loading: 'Lädt...',
        active: 'Aktive EDID',
        activeUnknown:
          'NanoKVM hat seit dem Start keine EDID geschrieben, daher ist die Kennung, die der Host sieht, unbekannt.',
        appliedAt: 'Angewendet am {{time}}',
        download: 'Herunterladen',
        downloadBackup: 'Vorherige herunterladen',
        preset: 'Monitorvorlage',
        presetPlaceholder: 'Monitor auswählen',
        upload: 'Hochladen',
        selected: 'Ausgewählte EDID',
        errors: 'Fehler',
        warnings: 'Warnungen',
        info: 'Informationen',
        unknownMonitor: 'Unbekannter Monitor',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Audio',
        audioNo: 'Kein Audio',
        extensionBlocks: 'Erweiterungsblöcke: {{blocks}}',
        apply: 'Anwenden',
        applyTitle: 'Diese EDID anwenden?',
        before: 'Aktuell',
        after: 'Neu',
        hdmiNotice:
          'Die Videoaufnahme stoppt, während die EDID geschrieben wird, und startet danach von selbst wieder.',
        powerCycleNotice:
          'Dieses Gerät muss physisch von der Stromversorgung getrennt und wieder angeschlossen werden, damit die neue EDID wirksam wird.',
        powerCycleUnverified:
          'Der Schreibvorgang wurde nicht verifiziert, der Videochip behält also, was jetzt darin steht, bis dieses Gerät physisch vom Strom getrennt und wieder angeschlossen wird.',
        applied: 'EDID angewendet und überprüft.',
        applyFailed: 'Das Anwenden der EDID ist fehlgeschlagen.',
        busy: 'Der Videochip war belegt. Bitte erneut versuchen.',
        unsupported: 'Dieses Gerät unterstützt das Ändern der EDID nicht.',
        toolMissing: 'Das EDID-Werkzeug fehlt in dieser Firmware.',
        noAudio:
          'Diese EDID meldet kein Audio, daher sendet der Host möglicherweise keinen Ton mehr.',
        oldVersion: 'Diese EDID verwendet eine ältere Version als 1.4.',
        interlaced: 'Das bevorzugte Timing ist im Zeilensprungverfahren.',
        tooLarge:
          'Das bevorzugte Timing ist größer als 1920x1080 bei 60 Hz und damit mehr, als NanoKVM aufnehmen kann.',
        recovery: 'Wiederherstellung',
        recoveryNeeded:
          'Der letzte Schreibvorgang wurde nicht verifiziert, daher ist der EDID-Bereich des Video-Chips in einem unbekannten Zustand. Stellen Sie das werkseitige EDID wieder her, um einen bekannten Zustand herzustellen.',
        recoveryDesc:
          'Schreibt ein bekanntes EDID zurück auf den Video-Chip, wenn ein angewendetes EDID den Host ohne Bild zurückgelassen hat.',
        restoreFactory: 'Werks-EDID wiederherstellen',
        restoreBackup: 'Vorheriges EDID wiederherstellen',
        restoreTitle: 'Dieses EDID wiederherstellen?',
        restoreFactoryTarget: 'Das werkseitige EDID, mit dem NanoKVM ausgeliefert wird.',
        restoreBackupTarget: 'Die neueste Sicherung, angewendet am {{time}}.',
        restoreNotice:
          'Eine Wiederherstellung wird genauso geschrieben wie eine Anwendung und hat dieselben Folgen.',
        restored: 'EDID wiederhergestellt und verifiziert.',
        restoreFailed: 'Die Wiederherstellung des EDID ist fehlgeschlagen.',
        mismatchTitle: 'Geschrieben und zurückgelesen',
        mismatchWritten: 'Geschrieben',
        mismatchRead: 'Zurückgelesen',
        restoreOkBtn: 'Wiederherstellen',
        hardware: 'Erkannte Hardware: {{hardware}}',
        hardwareUnknown: 'Unbekannt',
        confirmWord: 'ANWENDEN',
        confirmPrompt: 'Geben Sie {{word}} ein, um die Schaltfläche zum Anwenden zu aktivieren.',
        okBtn: 'Anwenden',
        cancelBtn: 'Abbrechen'
      },
      presentation: {
        title: 'USB-Darstellung',
        loading: 'Lädt...',
        current: 'Aktuelle USB-Darstellung',
        noProfile: 'Kein Profil angewendet',
        linked: 'Verknüpfte Funktionen',
        hostState: 'USB des Hosts',
        hostUnbound: 'Controller nicht gebunden',
        hdmiState: 'HDMI-Eingang',
        hdmiSignal: 'Signal vorhanden',
        hdmiUnreported: 'Noch keine Meldung der Videoaufnahme',
        endpoints: 'Endpoints',
        fifos: 'FIFO-Slots',
        pending: 'Ausstehende Änderungen',
        pendingEdits: 'Nicht gespeicherte Änderungen an der Identität',
        pendingProfile: '{{profile}} ist ausgewählt, aber nicht angewendet',
        pendingNone: 'Keine',
        lastApply: 'Letzte Anwendung',
        applyFailed: 'Fehlgeschlagen bei {{profile}} am {{time}}',
        applyClean: 'Kein Fehler verzeichnet',
        lastKnownGood: 'Letzter bekannter guter Stand',
        rollbackTarget: 'Ziel des Rollbacks',
        rollbackNone: 'Keines',
        powerCyclePending:
          'Der Controller wurde dem Host entzogen. Schalten Sie den angeschlossenen Rechner aus und wieder ein, um das Gerät zurückzubekommen.',
        rollback: 'Zurückrollen',
        rollbackTitle: 'Auf {{profile}} zurückrollen?',
        rollbackDesc: 'Das Gadget wird neu enumeriert; USB-Funktionen fallen kurz aus.',
        profile: 'USB-Profil',
        builtIn: 'integriert',
        descriptors: 'Deskriptoren',
        imported: 'importiert',
        clone: 'Klonen',
        cloneTitle: 'Dieses Profil klonen',
        cloneToEdit:
          'Integrierte Profile bleiben schreibgeschützt. Klonen Sie dieses Profil, um seine Identität zu bearbeiten.',
        profileName: 'Profilname',
        profileNameHint: 'Kleinbuchstaben, Ziffern, Punkte, Unterstriche und Bindestriche.',
        import: 'Paket importieren',
        export: 'Paket exportieren',
        delete: 'Löschen',
        deleteTitle: 'Dieses Profil löschen?',
        deleteDesc: '{{profile}} vom NanoKVM löschen.',
        identity: 'USB-Identität',
        preset: 'Vorlage für die Identität',
        presetPlaceholder: 'Identität von einem bekannten Gerät übernehmen',
        presetHint:
          'Eine Vorlage füllt die Vendor-ID, die Product-ID und die beiden Namensfelder. Deskriptoren bringt sie nicht mit.',
        presetSource: 'Identität wie in {{source}} verzeichnet',
        vendorId: 'Vendor-ID',
        foreignVendor: 'Diese Vendor-ID gehört einem anderen Hersteller',
        productId: 'Product-ID',
        bcdUSB: 'USB-Version',
        bcdDevice: 'Geräteversion',
        manufacturer: 'Hersteller',
        product: 'Produkt',
        serial: 'Seriennummer',
        configuration: 'Konfigurationsstring',
        hidLayout: 'HID-Geräte',
        hidRoleKeyboard: 'Tastatur',
        hidRoleRelative: 'Maus (relativ)',
        hidRoleAbsolute: 'Zeiger (absolut)',
        hidOff: 'Nicht vorhanden',
        hidInterface: 'Schnittstelle {{index}}',
        hidBootKeyboardShared:
          'Die Tastatur teilt sich eine Schnittstelle und bietet deshalb keinen Report im Boot-Protokoll mehr an. Manche BIOS- und UEFI-Setups sehen sie dann nicht.',
        functions: 'Funktionen',
        descriptorAssets: 'Gespeicherte Deskriptor-Dateien: {{count}}',
        endpointUse:
          'IN {{inUse}} belegt, {{inFree}} frei; OUT {{outUse}} belegt, {{outFree}} frei',
        apply: 'Anwenden',
        applyTitle: 'Dieses USB-Profil anwenden?',
        applyDesc: 'NanoKVM zeigt dem angeschlossenen Rechner {{profile}}.',
        reconnect:
          'Tastatur, Maus und andere USB-Funktionen werden kurz getrennt, während das Gadget neu gebunden wird.',
        applyLinks: 'Verknüpft: {{functions}}',
        applyRemoves: 'Entfernt: {{functions}}',
        applyNoHid:
          'Nach dieser Anwendung bleibt keine HID-Funktion übrig. Tastatur und Maus funktionieren dann nicht mehr.',
        applyRollback: 'Eine fehlgeschlagene Anwendung kehrt zu {{profile}} zurück.',
        recoveryPowerCycle:
          'Kein HID übersteht diese Anwendung; ein Host, der nicht mehr reagiert, lässt sich dann nur noch durch Aus- und Einschalten zurückholen.',
        recoveryReboot:
          'Eine Schnittstelle verschwindet aus dem Verbundgerät; der Host braucht unter Umständen einen Neustart, um den Rest neu zu binden.',
        recoveryHdmiReset:
          'Eine Videofunktion wird neu aufgebaut, dadurch setzt sich die dahinterliegende Aufnahmekette zurück.',
        recoveryReconnect: 'Der Host enumeriert das Gerät neu; USB-Funktionen fallen kurz aus.',
        cancel: 'Abbrechen',
        noFunctions: 'Keine verknüpften Funktionen',
        loadFailed: 'Die Darstellungsprofile konnten nicht geladen werden',
        operationFailed: 'Die Aktion an der USB-Darstellung ist fehlgeschlagen'
      },
      passthrough: {
        title: 'USB-Passthrough',
        loading: 'Wird geladen...',
        mode: 'Modus',
        hybrid: 'Hybrid',
        exact: 'Exakt',
        hybridDesc: 'Behält Boot-Tastatur und relative Maus, für kompatible Geräte.',
        exactDesc: 'Ersetzt jede USB-Funktion des NanoKVM durch das importierte Gerät.',
        hybridWarning: 'Hybrid hält Tastatur und relative Maus verfügbar',
        hybridWarningDesc:
          'Speicher, USB-Netzwerk und der absolute Zeiger werden getrennt, solange die importierte Funktion aktiv ist.',
        hidWarning: 'Ein Passthrough gibt Tastatur, Maus und virtuelle Medien ab',
        hidWarningDesc:
          'NanoKVM hat nur einen USB-Device-Controller, und der Proxy braucht ihn ganz. Während einer Sitzung sieht der entfernte Host deshalb das durchgereichte Gerät statt Tastatur, Maus und virtuellen Medien des NanoKVM. Sie kommen von selbst zurück, sobald die Sitzung beendet wird. Diese Weboberfläche ist nicht betroffen, Sie können eine Sitzung also jederzeit auf dieser Seite beenden.',
        hidWarningSafeDesc:
          'NanoKVM hat nur einen USB-Device-Controller, und der Proxy braucht ihn ganz. Während einer Sitzung sieht der entfernte Host deshalb das durchgereichte Gerät statt Tastatur, Maus und virtuellen Medien des NanoKVM. Sie kommen zurück, sobald die Sitzung beendet wird.',
        isoLabel: 'Isochrone Transfers erlauben',
        isoHint:
          'Lässt Webcams, Mikrofone und andere Streaming-Geräte durch. Niemand hat gemessen, was diese Hardware trägt.',
        isoWarning:
          'Isochrones Streaming ist hier unerprobt und kann Tastatur und Maus halten, bis Sie die Sitzung beenden',
        info: {
          title: 'Hinweise',
          hybrid:
            'Der Hybrid-Modus hält Tastatur und relative Maus verfügbar. Speicher, USB-Netzwerk und der absolute Zeiger werden getrennt, solange das importierte Gerät aktiv ist.',
          exact:
            'Der exakte Modus ersetzt jede USB-Funktion des NanoKVM durch das importierte Gerät. Tastatur, Maus und virtuelle Medien kommen von selbst zurück, sobald die Sitzung endet.',
          udc: 'NanoKVM hat nur einen USB-Device-Controller, und der Proxy braucht ihn ganz. Deshalb verschwinden die Funktionen oben für die Dauer einer Sitzung.',
          web: 'Diese Weboberfläche ist nicht betroffen, Sie können eine Sitzung also jederzeit auf dieser Seite beenden.',
          network:
            'Starten Sie den Passthrough über Ethernet oder WLAN. Ein Start aus dem USB-Netzwerk des NanoKVM wird abgelehnt, weil diese Verbindung verschwinden würde.',
          iso: 'Webcams, Mikrofone und andere isochrone Geräte werden abgelehnt, solange Sie isochrone Übertragungen nicht erlauben. Dieser Weg funktioniert, wurde auf dieser Hardware aber nie gemessen: Behandeln Sie den Durchsatz als unbekannt.',
          camera:
            'Kamera und Mikrofon des Browsers unter Geräte bleiben der erprobte Weg, dem Zielrechner eines zu geben.'
        },
        session: 'Sitzung',
        activeDesc: 'Ein Gerät ist importiert und der Proxy hält den USB-Controller.',
        inactiveDesc:
          'Es läuft keine Sitzung. Tastatur, Maus und virtuelle Medien funktionieren normal.',
        device: 'Gerät',
        busId: 'Bus-ID',
        speed: 'Geschwindigkeit',
        exporter: 'Exporter',
        local: 'Importiert als',
        localValue: 'Bus {{bus}}, Adresse {{address}}',
        udc: 'USB-Controller',
        pid: 'Proxy-PID',
        startedAt: 'Gestartet',
        isoDevice:
          'Dieses Gerät streamt über isochrone Endpunkte, was auf dieser Hardware nie gemessen wurde',
        exporterLabel: 'Adresse des Exporters',
        exporterHint:
          'Host und Port, die NanoKVM anwählt. Über den Tunnel unten ist das {{exporter}}.',
        busIdLabel: 'Bus-ID auf Ihrem Rechner',
        busIdHint: 'Die busid, die usbip list -l für das Gerät ausgibt, zum Beispiel {{example}}.',
        start: 'Passthrough starten',
        stop: 'Passthrough beenden',
        startTitle: 'USB-Passthrough starten?',
        startDevice: 'NanoKVM importiert {{busId}} von {{exporter}}.',
        startHid:
          'USB-Tastatur, Maus und virtuelle Medien funktionieren für die Dauer der Sitzung nicht und arbeiten von selbst wieder, sobald Sie sie beenden.',
        startIso:
          'Webcams und andere isochrone Geräte brauchen den isochronen Schalter, bevor Sie starten.',
        startWeb:
          'Diese Weboberfläche läuft weiter, Sie können die Sitzung also jederzeit auf dieser Seite beenden.',
        startNetwork:
          'Nutzen Sie diese Seite über Ethernet oder WLAN. Ein Start aus dem USB-Netzwerk des NanoKVM wird abgelehnt, weil diese Verbindung verschwinden würde.',
        okBtn: 'Starten',
        cancelBtn: 'Abbrechen',
        instructions: 'Auf Ihrem eigenen Rechner',
        instructionsDesc:
          'Es gibt bewusst keinen Client-Agenten zu installieren. Führen Sie diese normalen usbip-Befehle auf dem Rechner aus, an dem das Gerät hängt.',
        copyFailed: 'Kopieren fehlgeschlagen. Bitte den Befehl von Hand kopieren.',
        copyInsecure:
          'Diese Seite wird nicht über HTTPS ausgeliefert, deshalb blockiert der Browser das Kopieren. Den Befehl von Hand kopieren oder HTTPS unter Einstellungen, Netzwerk aktivieren.',
        directNote:
          'Ohne Tunnel muss usbipd in Ihrem Netz erreichbar sein und die Exporter-Adresse oben muss darauf zeigen. usbip überträgt das Gerät unverschlüsselt, der Tunnel ist deshalb vorzuziehen.',
        steps: {
          modprobe: {
            title: 'Treiber der Exportseite laden',
            desc: 'usbip-host erlaubt es Ihrem Kernel, ein lokales Gerät abzugeben. Es wird nicht automatisch geladen.'
          },
          list: {
            title: 'Gerät finden',
            desc: 'Listet jedes lokale Gerät mit busid und Hersteller:Produkt-Paar auf. Notieren Sie die busid des gewünschten Geräts.'
          },
          bind: {
            title: 'An usbip binden',
            desc: 'Nimmt das Gerät seinem normalen Treiber weg; es funktioniert auf diesem Rechner erst wieder nach dem Lösen der Bindung.'
          },
          serve: {
            title: 'Bereitstellen',
            desc: 'usbipd bleibt im Vordergrund und wartet darauf, dass NanoKVM das Gerät importiert.',
            notice:
              'Das normale usbipd hat keine Option für eine Lauschadresse und bindet an alle Schnittstellen. Halten Sie Port {{port}} in Ihrer Firewall geschlossen, damit ihn nur der Tunnel unten erreicht.'
          },
          tunnel: {
            title: 'Auf NanoKVM zeigen lassen',
            desc: 'Ein umgekehrter SSH-Tunnel: Port {{port}} auf dem Loopback des NanoKVM wird zum usbipd auf diesem Rechner. Lassen Sie ihn für die ganze Sitzung laufen.'
          },
          exporter: {
            title: 'Das als Exporter eintragen',
            desc: 'Tragen Sie das oben in das Feld für den Exporter ein, geben Sie die Bus-ID an und starten Sie die Sitzung.'
          },
          unbind: {
            title: 'Gerät zurückgeben',
            desc: 'Nach dem Ende der Sitzung gibt das Gerät damit an seinen normalen Treiber auf diesem Rechner zurück.'
          }
        }
      },
      mcp: {
        title: 'MCP-Dienst',
        service: 'MCP-Fernsteuerung',
        serviceDesc:
          'Vertrauenswürdigen MCP-Clients erlauben, Tastatur und Maus zu steuern und Bildschirmfotos aufzunehmen',
        securityWarning:
          'Jeder mit diesem API-Schlüssel kann den entfernten Host steuern und dessen Bildschirm sehen. Verwenden Sie HTTPS und aktivieren Sie den Dienst nur in vertrauenswürdigen Netzwerken.',
        endpoint: 'Endpunkt',
        apiKey: 'API-Schlüssel',
        regenerateConfirmTitle: 'MCP-API-Schlüssel neu generieren?',
        regenerateConfirmDesc: 'Der aktuelle Schlüssel funktioniert dann sofort nicht mehr.',
        enableConfirmTitle: 'Externe MCP-Steuerung aktivieren?',
        enableConfirmDesc:
          'Durch Aktivieren von MCP wird PicoClaw gestoppt und jede aktive PicoClaw-Sitzung geschlossen.',
        failed: 'MCP-Aktion fehlgeschlagen',
        copyFailed: 'Kopieren fehlgeschlagen. Bitte manuell kopieren.',
        copyInsecure:
          'Diese Seite wird nicht über HTTPS ausgeliefert, deshalb blockiert der Browser das Kopieren. Manuell kopieren oder HTTPS unter Einstellungen, Netzwerk aktivieren.',
        okBtn: 'Bestätigen',
        cancelBtn: 'Abbrechen'
      },
      about: {
        title: 'Über NanoKVM',
        information: 'Informationen',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applikations-Version',
        applicationTip: 'NanoKVM Web Applikations-Version',
        image: 'Systemabbild-Version',
        imageTip: 'NanoKVM Systemabbild-Version',
        deviceKey: 'Geräteschlüssel',
        community: 'Community',
        hostname: 'Hostname',
        hostnameUpdated: 'Hostname aktualisiert. Neustarten um zu übernehmen.',
        ipType: {
          Wired: 'Kabel',
          Wireless: 'Drahtlos',
          Other: 'Andere'
        }
      },
      appearance: {
        title: 'Erscheinungsbild',
        display: 'Bildschirm',
        language: 'Sprache',
        languageDesc: 'Wählen Sie die Sprache für die Benutzeroberfläche aus',
        webTitle: 'Web Titel',
        webTitleDesc: 'Passen Sie den Web-Seite Titel an',
        favicon: 'Favicon',
        faviconDesc: 'Symbol im Browser-Tab anpassen',
        faviconPlaceholder: 'Bild-URL',
        faviconUpload: 'Hochladen',
        faviconReset: 'Zurücksetzen',
        faviconCustom: 'Eigenes Symbol',
        faviconBoot: 'Symbol aus /boot/logo.ico',
        faviconDefault: 'Standardsymbol',
        faviconOverridesBoot: 'Überschreibt /boot/logo.ico',
        faviconErrUrl: 'Geben Sie eine http:// oder https:// Bildadresse ein',
        faviconErrFetch: 'Das Gerät konnte das Bild nicht herunterladen',
        faviconErrLarge: 'Bild ist zu groß. Das Limit liegt bei 256 KB',
        faviconErrType: 'Nicht unterstütztes Bild. Verwenden Sie .ico, .png, .svg, .gif oder .jpg',
        faviconErrSave: 'Symbol konnte nicht gespeichert werden',
        menuBar: {
          title: 'Menüleiste',
          mode: 'Anzeigemodus',
          modeDesc: 'Menüleiste auf dem Bildschirm anzeigen',
          modeOff: 'Aus',
          modeAuto: 'Automatisch ausblenden',
          modeAlways: 'Immer sichtbar',
          keyboardLedStatus: 'Tastensperren-Anzeigen',
          keyboardLedStatusDesc:
            'Num-Lock-, Feststell- und Rollen-Status des Remote-Computers anzeigen',
          icons: 'Untermenüsymbole',
          iconsDesc: 'Untermenüsymbole in der Menüleiste anzeigen'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Tastensperren-Status der Remote-Tastatur',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num-Taste',
        numLockShort: 'Num',
        capsLock: 'Feststelltaste',
        capsLockShort: 'Fest',
        scrollLock: 'Rollen-Taste',
        scrollLockShort: 'Roll',
        on: 'Ein',
        off: 'Aus',
        unknown: 'Unbekannt'
      },
      device: {
        title: 'Gerät',
        oled: {
          title: 'OLED',
          description: 'Schalte OLED Bildschirm aus nach',
          0: 'Nie',
          15: '15 Sek',
          30: '30 Sek',
          60: '1 Min',
          180: '3 Min',
          300: '5 Min',
          600: '10 Min',
          1800: '30 Min',
          3600: '1 Stunde'
        },
        ssh: {
          description: 'Aktiviere entfernten SSH-Zugang',
          tip: 'Setzten Sie ein starkes Passwort vor dem aktivieren (Konto - Passwort ändern)'
        },
        advanced: 'Erweiterte Einstellungen',
        swap: {
          title: 'Swap',
          disable: 'Deaktivieren',
          description: 'Grösse der Swap-Datei festlegen',
          tip: 'Das Aktivieren dieser Funktion kann die Lebensdauer Ihrer SD-Karte verkürzen!'
        },
        mouseJiggler: {
          title: 'Mausaktivitäts-Simulator',
          description: 'Verhindert, dass der remote Host in den Energiesparmodus wechselt',
          disable: 'Deaktivieren',
          absolute: 'Absoluter Modus',
          relative: 'Relativer Modus'
        },
        mdns: {
          description: 'mDNS-Erkennungsdienst aktivieren',
          tip: 'Deaktivieren Sie den Dienst, wenn Sie ihn nicht benötigen'
        },
        hdmi: {
          description: 'HDMI/Monitor-Ausgabe aktivieren',
          idleTimeoutTitle: 'Zeitlimit für inaktive Aufnahme',
          idleTimeoutDescription:
            'HDMI-Aufnahme stoppen, wenn keine aktiven Zuschauer vorhanden sind für',
          minutes: 'Min.'
        },
        autostart: {
          title: 'Autostart-Skripteinstellungen',
          description: 'Skripte verwalten, die beim Systemstart automatisch ausgeführt werden',
          new: 'Neu',
          deleteConfirm: 'Möchten Sie diese Datei wirklich löschen?',
          yes: 'Ja',
          no: 'Nein',
          scriptName: 'Name des Autostart-Skripts',
          scriptContent: 'Inhalt des Autostart-Skripts',
          settings: 'Einstellungen'
        },
        hidOnly: 'HID-Only Mode',
        hidOnlyDesc:
          'Hören Sie auf, virtuelle Geräte zu emulieren, und behalten Sie nur die grundlegende HID-Steuerung bei',
        disk: 'Virtuelle Festplatte',
        diskDesc: 'Binde das virtuelle U-Laufwerk an den entfernten Host',
        rebindNotice:
          'Das Umschalten dieses Schalters meldet das USB-Gerät neu an, das Zielsystem verliert dabei kurz seine virtuellen Geräte und sein USB-Netzwerk.',
        media: {
          title: 'Kamera- und Mikrofonplätze',
          desc: 'Legen Sie fest, welche Mediengeräte Browser belegen dürfen. Das Endpunkt-Budget wird beim Anwenden des USB-Profils geprüft. Beim Speichern wird das Gerät neu erkannt und verbundene Browser werden getrennt.',
          cameras: 'Kameras',
          microphones: 'Mikrofone',
          name: 'Name',
          namePlaceholder: 'Wird auf dem Zielrechner angezeigt',
          addCamera: 'Kamera hinzufügen',
          addMicrophone: 'Mikrofon hinzufügen',
          remove: 'Entfernen',
          cameraDefault: 'Kamera {{index}}',
          microphoneDefault: 'Mikrofon {{index}}',
          nameRequired: 'Jeder Platz braucht einen Namen.',
          budgetHint:
            'Die sechs USB-IN-Endpunkte sind eine feste Hardwaregrenze. Lege Tastatur, Maus und Absolutzeiger unter USB-Darstellung auf eine HID-Schnittstelle, oder schalte hier die virtuelle Festplatte oder unter Netzwerk den USB-Netzwerkadapter ab.',
          unsupported:
            'Dieser Kernel kann Mediengeräte nicht benennen, deshalb zeigen Hosts den Standardnamen.',
          save: 'Plätze speichern',
          disconnect: 'Trennen',
          disconnectAll: 'Alle Quellen trennen',
          limit: 'Kamera- und Mikrofonplätze dürfen zusammen höchstens acht ergeben.',
          failed: 'Die Medienplätze konnten nicht aktualisiert werden.'
        },
        reboot: 'Neustarten',
        rebootDesc: 'Sind Sie sicher dass Sie NanoKVM neustarten möchten?',
        okBtn: 'Ja',
        cancelBtn: 'Nein'
      },
      network: {
        title: 'Netzwerk',
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi konfigurieren',
          apMode: 'AP-Modus ist aktiviert, verbinden Sie sich per QR-Code mit dem Wi-Fi',
          connect: 'Wi-Fi verbinden',
          connectDesc1: 'Bitte geben Sie die Netzwerk-SSID und das Passwort ein',
          connectDesc2: 'Bitte geben Sie das Passwort ein, um diesem Netzwerk beizutreten',
          disconnect: 'Möchten Sie die Netzwerkverbindung wirklich trennen?',
          failed: 'Verbindung fehlgeschlagen, bitte erneut versuchen.',
          ssid: 'Name',
          password: 'Passwort',
          joinBtn: 'Verbinden',
          confirmBtn: 'OK',
          cancelBtn: 'Abbrechen'
        },
        tls: {
          description: 'HTTPS-Protokoll aktivieren',
          tip: 'Hinweis: Die Verwendung von HTTPS kann die Latenz erhöhen, besonders im MJPEG-Videomodus.'
        },
        usb: {
          title: 'USB-Netzwerkadapter',
          desc: 'Gibt dem gesteuerten Computer eine Netzwerkkarte über USB',
          type: 'Adaptertyp',
          typeDesc: 'NCM für moderne Systeme, RNDIS für ältere Windows-Systeme'
        },
        bridge: {
          title: 'Adapter verbunden mit',
          lan: 'Ihr Netzwerk',
          kvmOnly: 'Nur NanoKVM',
          lanDesc:
            'Der Computer kommt über den Ethernet-Port des NanoKVM in Ihr Netzwerk und erhält eine eigene Adresse vom Router.',
          kvmOnlyDesc:
            'Der Computer erhält seine Adresse vom NanoKVM und erreicht das NanoKVM, aber nichts dahinter.',
          loading: 'Wird geladen...',
          state: 'Status',
          states: {
            disabled: 'Nur NanoKVM',
            enabled: 'Ihr Netzwerk',
            rolledBack: 'Zurückgesetzt',
            failed: 'Fehlgeschlagen',
            pending: 'Läuft'
          },
          uplink: 'Uplink',
          ports: 'Ports',
          up: 'verbunden',
          down: 'getrennt',
          noLink: 'kein Link',
          enableTitle: 'Computer mit Ihrem Netzwerk verbinden?',
          disableTitle: 'Computer auf nur NanoKVM beschränken?',
          reconnect:
            'Die Verwaltungsverbindung wird kurz getrennt und neu aufgebaut, während die Adresse umzieht.',
          rollback:
            'Schlägt die Überprüfung fehl, wird die vorherige Konfiguration automatisch wiederhergestellt.',
          enableBtn: 'Mit meinem Netzwerk verbinden',
          disableBtn: 'Nur NanoKVM',
          cancelBtn: 'Abbrechen',
          interrupted:
            'Die Verbindung wurde während der Anwendung unterbrochen. Der aktuelle Status wird erneut geprüft.',
          pendingNotice:
            'Eine Änderung der Brücke läuft noch oder wurde vor dem Abschluss unterbrochen.',
          revert: 'Vorherige Konfiguration wiederherstellen',
          rolledBackNotice:
            'Die letzte Änderung wurde zurückgenommen und die vorherige Konfiguration wiederhergestellt.',
          verifyFailed: 'Überprüfung fehlgeschlagen: {{gates}}',
          gates: {
            address: 'Adresse',
            gateway: 'Gateway',
            inbound: 'eingehende Verbindung'
          },
          inboundWeak:
            'Die Prüfung eingehender Verbindungen gelang nur, weil NanoKVM sich mit sich selbst verbunden hat. Das belegt, dass der Webdienst lauscht und lokal erreichbar ist, nicht aber, dass eine Anfrage aus dem Netzwerk ankommt.',
          noCarrier:
            'Kein Link an {{port}}. Die Brücke hat keinen Weg ins Netzwerk, solange kein Kabel steckt.',
          loop: 'Der Router wird auch an {{port}} gelernt, dieser Port ist also ein zweiter Weg in dasselbe Netzwerk. Spanning Tree ist aus, hier wird die Schleife also von nichts aufgebrochen: Trennen Sie einen der beiden Wege.',
          failedNotice:
            'Die letzte Änderung konnte nicht rückgängig gemacht werden. NanoKVM ist möglicherweise nur über den WLAN-AP oder eine serielle Konsole erreichbar.'
        },
        dns: {
          title: 'DNS',
          description: 'DNS-Server für NanoKVM konfigurieren',
          mode: 'Modus',
          dhcp: 'DHCP',
          manual: 'Manuell',
          add: 'DNS hinzufügen',
          save: 'Speichern',
          invalid: 'Bitte geben Sie eine gültige IP-Adresse ein',
          noDhcp: 'Derzeit ist kein DHCP-DNS verfügbar',
          saved: 'DNS-Einstellungen gespeichert',
          saveFailed: 'DNS-Einstellungen konnten nicht gespeichert werden',
          unsaved: 'Ungespeicherte Änderungen',
          maxServers: 'Maximal {{count}} DNS-Server erlaubt',
          dnsServers: 'DNS-Server',
          dhcpServersDescription: 'DNS-Server werden automatisch per DHCP bezogen',
          manualServersDescription: 'DNS-Server können manuell bearbeitet werden',
          networkDetails: 'Netzwerkdetails',
          interface: 'Schnittstelle',
          ipAddress: 'IP-Adresse',
          subnetMask: 'Subnetzmaske',
          router: 'Router',
          none: 'Keine'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC-Server',
        description:
          'Erlaubt jedem VNC-Client, den entfernten Bildschirm zu sehen und Tastatur und Maus zu benutzen, mit Anmeldung über Ihr NanoKVM-Konto',
        port: 'Port',
        portDescription: 'Verbinden Sie sich mit diesem Port auf der NanoKVM-Adresse'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Speicher Optimierung',
          tip: 'Wenn die Speichernutzung das Limit überschreitet, wird die Speicherbereinigung aggressiver durchgeführt, um Speicher freizugeben. Es wird empfohlen, den Wert auf 75 MB zu setzen, wenn Tailscale verwendet wird. Ein Neustart von Tailscale ist erforderlich, damit die Änderung wirksam wird.'
        },
        swap: {
          title: 'Speicher austauschen',
          tip: 'Wenn die Probleme nach der Aktivierung der Speicheroptimierung weiterhin bestehen, versuchen Sie, den Swap-Speicher zu aktivieren. Dadurch wird die Größe der Auslagerungsdatei standardmäßig auf 256MB festgelegt, was unter „Einstellungen > Gerät“ angepasst werden kann.'
        },
        restart: 'Tailscale neu starten?',
        stop: 'Tailscale stoppen?',
        stopDesc: 'Von Tailscale abmelden und automatischen Start beim Booten deaktivieren.',
        loading: 'Lädt...',
        notInstall: 'Tailscale nicht gefunden! Bitte installieren.',
        install: 'Installieren',
        installing: 'Installiere',
        failed: 'Installation fehlgeschlagen',
        retry: 'Bitte Seite neu laden und erneut versuchen oder manuelle Installation versuchen.',
        download: 'Laden Sie das',
        package: 'Installations-Paket herunter',
        unzip: 'und entpacken Sie es',
        upTailscale: 'Tailscale nach /usr/bin/ auf NanoKVM hochladen',
        upTailscaled: 'Tailscaled nach /usr/bin/ auf NanoKVM hochladen',
        refresh: 'Aktuelle Seite neu laden',
        notRunning: 'Tailscale läuft nicht. Bitte starten Sie es, um fortzufahren.',
        run: 'Start',
        notLogin:
          'Das Gerät konnte noch nicht gefunden werden. Bitte melden Sie sich an und verknüpfen Sie dieses Gerät mit Ihrem Konto.',
        urlPeriod: 'Diese URL ist für 10 Minuten gültig',
        login: 'Anmelden',
        loginSuccess: 'Anmeldung erfolgreich',
        enable: 'Tailscale einschalten',
        deviceName: 'Geräte Name',
        deviceIP: 'Geräte IP',
        account: 'Konto',
        logout: 'Abmelden',
        logoutDesc: 'Möchten Sie sich wirklich abmelden?',
        uninstall: 'Tailscale deinstallieren',
        uninstallDesc: 'Sind Sie sicher, dass Sie Tailscale deinstallieren möchten?',
        okBtn: 'Ja',
        cancelBtn: 'Nein'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Lädt...',
        notInstall: 'Nicht installiert',
        notConfigured: 'Nicht konfiguriert',
        stopped: 'Gestoppt',
        running: 'Läuft',
        connected: 'Verbunden',
        error: 'Fehler',
        atBoot: 'startet beim Hochfahren',
        notAtBoot: 'startet nicht beim Hochfahren',
        arguments: 'Argumente',
        argumentsTip: 'Kommandozeilenargumente, die dem Dienst beim Start übergeben werden.',
        env: 'Umgebungsvariablen',
        envKey: 'Name',
        envValue: 'Wert',
        envAdd: 'Variable hinzufügen',
        envRemove: 'Entfernen',
        configured: 'Konfiguriert',
        save: 'Speichern',
        saved: 'Konfiguration gespeichert',
        start: 'Start',
        stop: 'Stoppen',
        restart: 'Neu starten',
        logs: 'Protokoll',
        logsEmpty: 'Noch keine Protokolleinträge',
        refresh: 'Aktualisieren',
        binary: 'Binärdatei',
        binaryShipped: 'Mit der Firmware geliefert',
        binaryCustom: 'Eigene Binärdatei',
        binaryUpload: 'Binärdatei hochladen',
        binaryRevert: 'Mitgelieferte Binärdatei wiederherstellen',
        binaryRevertDesc:
          'Die hochgeladene Binärdatei löschen und die Version aus der Firmware wiederherstellen?',
        serverWarning: 'Ein Server ohne Einschränkungen wirkt wie ein offener Proxy',
        noHealthSignal:
          'Dieser Dienst meldet keinen Zustand, daher weiß NanoKVM nur, dass der Prozess läuft, nicht ob der Tunnel verbunden ist.',
        memoryWarning:
          'Mehrere gleichzeitig laufende Fernzugriffsdienste können den Speicher erschöpfen',
        resources: 'Ressourcen',
        memory: {
          title: 'Speichergrenze',
          description:
            'Begrenzt den Go-Heap von newt ab dem nächsten Neustart auf {{limit}} MiB. Die eigene Grenze, nicht die von Tailscale; ausgeschaltet gilt der Go-Standard, GOGC=50 wirkt in beiden Fällen.',
          noRuntime:
            'wstunnel ist Rust: kein Garbage Collector und keine Heap-Grenze zum Setzen, und seine Worker-Threads richten sich ohnehin nach der einzelnen CPU des Geräts.',
          notApplicable: 'Nicht anwendbar'
        },
        swap: {
          title: 'Auslagerungsdatei',
          description:
            'Legt eine 256-MB-Auslagerungsdatei auf der SD-Karte an. Systemweit: derselbe Swap dient Tailscale, dem KVM-Server und allem anderen auf dem Gerät.'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nein'
      },
      update: {
        title: 'Nach Aktualisierungen suchen',
        queryFailed: 'Version konnte nicht abgefragt werden',
        updateFailed: 'Aktualisierung fehlgeschlagen. Bitte versuchen Sie es erneut.',
        isLatest: 'Sie haben bereits die aktuellste Version.',
        rebooting:
          'Der neue Kernel wird installiert und das Gerät startet neu. Das kann einige Minuten dauern; trennen Sie die Stromversorgung nicht.',
        kernelUpdate:
          'Dieses Update installiert Kernel {{version}}. Das Gerät startet neu und kehrt von selbst zum aktuellen Kernel zurück, falls der neue nicht hochkommt.',
        rolledBack:
          'Kernel {{version}} ist nicht gestartet; das Gerät ist auf den vorherigen Kernel zurückgefallen.',
        available: 'Eine Aktualisierung ist verfügbar. Möchten sie diese jetzt durchführen?',
        updating: 'Aktualisierung gestartet. Bitte warten...',
        confirm: 'Bestätigen',
        cancel: 'Abbrechen',
        preview: 'Vorab-Versionen',
        previewDesc: 'Erhalten Sie vorab Zugriff auf neue Funktionen und Verbesserungen',
        previewTip:
          'Bitte beachten Sie, dass Vorab-Versionen womöglich noch Fehler oder unvollständige Funktionen enthalten!',
        customServer: {
          title: 'Benutzerdefinierter Update-Server',
          desc: 'Online-Updates von einem angegebenen Server suchen und herunterladen',
          invalidUrl:
            'Geben Sie ein gültiges HTTP- oder HTTPS-Serververzeichnis ohne Abfrageparameter, Fragment oder latest.json ein.',
          loadFailed: 'Die Konfiguration des Update-Servers konnte nicht geladen werden.',
          saveFailed: 'Die Konfiguration des Update-Servers konnte nicht gespeichert werden.',
          saved: 'Die Konfiguration des Update-Servers wurde gespeichert.',
          save: 'Speichern',
          confirmTitle: 'Benutzerdefinierten Update-Server verwenden?',
          confirmDesc:
            'SHA-512 bestätigt lediglich, dass das Paket mit dem von diesem Server bereitgestellten Manifest übereinstimmt. Es beweist nicht, dass das Paket eine offizielle NanoKVM-Version ist. Ein fehlerhafter oder bösartiger Server kann das Gerät unbrauchbar machen, Datenverlust verursachen oder das System kompromittieren.',
          confirm: 'Trotzdem verwenden',
          previewDisabled:
            'Vorschau-Updates sind nicht verfügbar, solange ein benutzerdefinierter Update-Server aktiviert ist.'
        },
        offline: {
          kernelNotice:
            'Dieses Paket enthält einen Kernel. Er wird in den Reserveplatz geschrieben und das Gerät startet zum Test neu; kommt es nicht zurück, kehrt es von selbst zum aktuellen Kernel zurück.',
          kernelConfirm: 'Kernel installieren',
          kernelCancel: 'Abbrechen',
          title: 'Offline Aktualisierung',
          desc: 'Über lokales Installationspaket aktualisieren',
          upload: 'Hochladen',
          checksumPlaceholder: 'SHA-256-Prüfsumme (optional)',
          invalidChecksum: 'Die SHA-256-Prüfsumme muss 64 hexadezimale Zeichen enthalten.',
          checksumMismatch:
            'Die SHA-256-Überprüfung ist fehlgeschlagen. Das Paket ist möglicherweise beschädigt.',
          invalidName:
            'Ungültiges Dateinamenformat. Bitte laden Sie von den GitHub-Releases herunter.',
          updateFailed: 'Aktualisierung fehlgeschlagen. Bitte versuchen Sie es erneut.'
        }
      },
      account: {
        title: 'Konto',
        webAccount: 'Web Konto Name',
        role: 'Rolle',
        roles: {
          admin: 'Administrator',
          user: 'Benutzer'
        },
        password: 'Passwort',
        updateBtn: 'Ändern',
        logoutBtn: 'Abmelden',
        logoutDesc: 'Möchten Sie sich wirklich abmelden?',
        okBtn: 'Ja',
        cancelBtn: 'Nein',
        users: {
          title: 'Benutzer',
          create: 'Benutzer anlegen',
          enabled: 'Aktiviert',
          disabled: 'Deaktiviert',
          deviceOwner: 'Gerätebesitzer',
          resetPassword: 'Passwort zurücksetzen',
          delete: 'Löschen',
          deleteConfirm: 'Diesen Benutzer löschen und alle seine Sitzungen beenden?',
          created: 'Benutzer angelegt',
          deleted: 'Benutzer gelöscht',
          passwordUpdated: 'Passwort aktualisiert',
          loadFailed: 'Benutzer konnten nicht geladen werden',
          saveFailed: 'Benutzer konnte nicht gespeichert werden',
          deleteFailed: 'Benutzer konnte nicht gelöscht werden'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistent',
      empty: 'Öffnen Sie das Bedienfeld und starten Sie eine Aufgabe.',
      inputPlaceholder: 'Beschreiben Sie, was der PicoClaw tun soll',
      newConversation: 'Neues Gespräch',
      processing: 'Wird verarbeitet...',
      agent: {
        defaultTitle: 'Allgemeiner Assistent',
        defaultDescription: 'Allgemeine Chat-, Such- und Arbeitsbereichshilfe.',
        kvmTitle: 'Fernsteuerung',
        kvmDescription: 'Betreiben Sie den Remote-Host über NanoKVM.',
        switched: 'Agentenrolle gewechselt',
        switchFailed: 'Agentenrolle konnte nicht gewechselt werden'
      },
      send: 'Senden',
      cancel: 'Abbrechen',
      status: {
        connecting: 'Verbindung zum Gateway wird hergestellt...',
        connected: 'PicoClaw Sitzung verbunden',
        disconnected: 'PicoClaw Sitzung geschlossen',
        stopped: 'Stoppanforderung gesendet',
        runtimeStarted: 'PicoClaw Runtime gestartet',
        runtimeStartFailed: 'PicoClaw Runtime konnte nicht gestartet werden',
        runtimeStopped: 'PicoClaw Runtime gestoppt',
        runtimeStopFailed: 'PicoClaw Runtime konnte nicht gestoppt werden',
        controlSwitchedToMCP: 'Steuerung zum externen MCP-Dienst gewechselt'
      },
      connection: {
        runtime: {
          checking: 'Überprüfung',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime bereit',
          stopped: 'Runtime gestoppt',
          blockedByMCP: 'Externe MCP-Steuerung ist aktiv',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime nicht verfügbar',
          configError: 'Konfigurationsfehler'
        },
        transport: {
          connecting: 'Verbinden',
          connected: 'Verbunden',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Leerlauf',
          busy: 'Beschäftigt'
        }
      },
      message: {
        toolAction: 'Aktion',
        observation: 'Beobachtung',
        screenshot: 'Screenshot'
      },
      overlay: {
        locked: 'PicoClaw steuert das Gerät. Die manuelle Eingabe wird angehalten.'
      },
      control: {
        picoclaw: 'Gerätesteuerung: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Gerätesteuerung: externes MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Gerätesteuerung: aus',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Steuerung erteilen',
        release: 'Freigeben',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw-Steuerung erteilt',
        released: 'PicoClaw-Steuerung freigegeben',
        grantFailed: 'PicoClaw-Steuerung konnte nicht erteilt werden',
        releaseFailed: 'PicoClaw-Steuerung konnte nicht freigegeben werden',
        grantConfirmTitle: 'Gerätesteuerung zu PicoClaw wechseln?',
        grantConfirmDesc: 'Externe MCP-Geräteschreibvorgänge werden unterbrochen.'
      },
      install: {
        install: 'Installieren Sie PicoClaw',
        installing: 'Installation von PicoClaw',
        success: 'PicoClaw erfolgreich installiert',
        failed: 'Installation von PicoClaw fehlgeschlagen',
        uninstalling: 'Runtime wird deinstalliert...',
        uninstalled: 'Runtime erfolgreich deinstalliert.',
        uninstallFailed: 'Deinstallation fehlgeschlagen.',
        requiredTitle: 'PicoClaw ist nicht installiert',
        requiredDescription: 'Installieren Sie PicoClaw, bevor Sie die PicoClaw Runtime starten.',
        progressDescription: 'PicoClaw wird heruntergeladen und installiert.',
        stages: {
          preparing: 'Vorbereiten',
          downloading: 'Wird heruntergeladen',
          extracting: 'Extrahieren',
          verifying: 'Überprüfen',
          installing: 'Installiere',
          installed: 'Installiert',
          install_timeout: 'Zeitüberschreitung',
          install_failed: 'Fehlgeschlagen'
        }
      },
      model: {
        requiredTitle: 'Modellkonfiguration ist erforderlich',
        requiredDescription:
          'Konfigurieren Sie das PicoClaw-Modell, bevor Sie den PicoClaw-Chat verwenden.',
        docsTitle: 'Konfigurationshandbuch',
        docsDesc: 'Unterstützte Modelle und Protokolle',
        menuLabel: 'Modell konfigurieren',
        modelIdentifier: 'Modell-ID',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API-Schlüssel',
        apiKeyPlaceholder: 'API-Schlüssel des Modells eingeben',
        save: 'Speichern',
        saving: 'Speichern',
        saved: 'Modellkonfiguration gespeichert',
        saveFailed: 'Modellkonfiguration konnte nicht gespeichert werden',
        invalid: 'Modellkennung, API Base URL und API-Schlüssel sind erforderlich'
      },
      uninstall: {
        menuLabel: 'Deinstallieren',
        confirmTitle: 'Deinstallieren PicoClaw',
        confirmContent:
          'Sind Sie sicher, dass Sie PicoClaw deinstallieren möchten? Dadurch werden die ausführbare Datei und alle Konfigurationsdateien gelöscht.',
        confirmOk: 'Deinstallieren',
        confirmCancel: 'Abbrechen'
      },
      history: {
        title: 'Verlauf',
        loading: 'Sitzungen werden geladen...',
        emptyTitle: 'Noch keine Historie',
        emptyDescription: 'Frühere PicoClaw-Sitzungen werden hier angezeigt.',
        loadFailed: 'Der Sitzungsverlauf konnte nicht geladen werden',
        deleteFailed: 'Sitzung konnte nicht gelöscht werden',
        deleteConfirmTitle: 'Sitzung löschen',
        deleteConfirmContent: 'Sind Sie sicher, dass Sie „{{title}}“ löschen möchten?',
        deleteConfirmOk: 'Löschen',
        deleteConfirmCancel: 'Abbrechen',
        messageCount_one: '{{count}} Nachricht',
        messageCount_other: '{{count}} Nachrichten',
        messageCount: '{{count}} Nachrichten'
      },
      config: {
        startRuntime: 'PicoClaw starten',
        stopRuntime: 'PicoClaw stoppen'
      },
      start: {
        enableConfirmTitle: 'Steuerung zu PicoClaw wechseln?',
        enableConfirmDesc: 'Beim Starten von PicoClaw wird der externe MCP-Dienst deaktiviert.',
        enableConfirmOk: 'PicoClaw starten',
        enableConfirmCancel: 'Abbrechen',
        title: 'PicoClaw starten',
        description:
          'Starten Sie die Runtime, um mit der Verwendung des PicoClaw-Assistenten zu beginnen.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Wir sind auf ein Problem gestossen',
      refresh: 'Neuladen'
    },
    fullscreen: {
      toggle: 'Vollbild ein/aus'
    },
    menu: {
      collapse: 'Menu einblenden',
      expand: 'Menu verbergen'
    }
  }
};

export default de;
