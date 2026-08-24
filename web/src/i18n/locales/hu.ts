const hu = {
  translation: {
    head: {
      desktop: 'Távoli Asztal',
      login: 'Bejelentkezés',
      changePassword: 'Jelszó megváltoztatása',
      terminal: 'Terminál',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Bejelentkezés',
      placeholderUsername: 'Adja meg a felhasználónevet',
      placeholderPassword: 'Adja meg a jelszót',
      placeholderCurrentPassword: 'Jelenlegi jelszó',
      placeholderPassword2: 'Adja meg újra a jelszót',
      noEmptyUsername: 'A felhasználónév nem lehet üres',
      noEmptyPassword: 'A jelszó nem lehet üres',
      passwordLength: 'A jelszónak 8 és 72 karakter között kell lennie',
      noAccount:
        'Nem sikerült megszerezni a felhasználói információkat, frissítse az oldalt vagy állítsa vissza a jelszót',
      invalidUser: 'Érvénytelen felhasználónév vagy jelszó',
      locked: 'Túl sok bejelentkezés, kérjük, próbálja újra később',
      globalLocked: 'A rendszer védelem alatt áll, próbálkozzon újra később',
      error: 'Váratlan hiba',
      invalidCurrentPassword: 'A jelenlegi jelszó helytelen',
      changePassword: 'Jelszó megváltoztatása',
      changePasswordDesc:
        'Az eszköz biztonsága érdekében módosítsa a webes bejelentkezési jelszót.',
      differentPassword: 'A jelszavak nem egyeznek',
      illegalUsername: 'A felhasználónév illegális karaktereket tartalmaz',
      illegalPassword: 'A jelszó illegális karaktereket tartalmaz',
      forgetPassword: 'Jelszó-emlékeztető',
      ok: 'Ok',
      cancel: 'Mégse',
      loginButtonText: 'Bejelentkezés',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'A részletes lépésekért tekintse meg ezt a dokumentumot:',
        reset3: 'Alapértelmezett webes fiók:',
        reset4: 'Alapértelmezett SSH-fiók:',
        change1: 'Vegye figyelembe, hogy ez a művelet a következő jelszavakat módosítja:',
        change2: 'Webes bejelentkezési jelszó',
        change3: 'Rendszer root jelszava (SSH bejelentkezési jelszó)',
        change4: 'A jelszavak visszaállításához tartsa lenyomva a BOOT gombot a NanoKVM-en.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Wi-Fi beállítása a NanoKVM-hez',
      success: 'Please check the network status of NanoKVM and visit the new IP address.',
      failed: 'A művelet sikertelen, próbálja újra.',
      invalidMode:
        'Az aktuális mód nem támogatja a hálózat beállítását. Kérjük, lépjen az eszközére, és engedélyezze a Wi-Fi konfigurációs módot.',
      confirmBtn: 'Ok',
      finishBtn: 'Kész',
      ap: {
        authTitle: 'Hitelesítés szükséges',
        authDescription: 'A folytatáshoz adja meg a AP jelszót',
        authFailed: 'Érvénytelen AP jelszó',
        passPlaceholder: 'AP jelszót',
        verifyBtn: 'Ellenőrizze'
      }
    },
    screen: {
      scale: 'Skála',
      title: 'Képernyő',
      video: 'Videó mód',
      videoDirectTips:
        'Engedélyezze az HTTPS elemet a "Beállítások > Eszköz" menüpontban ennek a módnak a használatához',
      resolution: 'Felbontás',
      auto: 'Automatikus',
      autoTips:
        'Bizonyos felbontások esetén képernyőszakadás vagy egéreltolódás léphet fel. Fontolja meg a távoli gép felbontásának módosítását vagy az automatikus mód kikapcsolását.',
      fps: 'FPS',
      customizeFps: 'Testreszabás',
      quality: 'Minőség',
      qualityLossless: 'Veszteségmentes',
      qualityHigh: 'Magas',
      qualityMedium: 'Közepes',
      qualityLow: 'Alacsony',
      frameDetect: 'Képkocka-figyelés',
      frameDetectTip:
        'Elemzi a képkockák közötti különbségeket. A videó stream küldése leáll, ha a távoli gép képernyőjén nem történik változás.',
      resetHdmi: 'HDMI visszaállítása',
      mixedH264: {
        title: 'H.264 adatfolyam-ütközés',
        description:
          'Az H.264 Direct és az H.264 WebRTC egyszerre van használatban. Ez képtörést vagy sérült videót okozhat. Csak egy H.264 módot használjon.'
      },
      webrtcConnectionFailed: {
        title: 'A WebRTC-kapcsolat sikertelen',
        description: 'Ellenőrizze a hálózati kapcsolatot, vagy váltson videómódot.'
      },
      captureStatus: {
        hdmiError: 'HDMI-képernyőhiba',
        unsupportedResolution: 'A jelenlegi felbontás nem támogatott',
        retrieving: 'Kép lekérése...',
        changingResolution: 'Felbontás váltása...',
        updateFailed: 'A kép jelenleg nem frissíthető',
        videoError: 'Videómegjelenítési hiba',
        noHdmi: 'Nem észlelhető HDMI-jel',
        unavailable: 'A kép jelenleg nem jeleníthető meg'
      }
    },
    keyboard: {
      title: 'Billentyűzet',
      paste: 'Beillesztés',
      tips: 'Csak a szabványos billentyűzet betűi és szimbólumai támogatottak',
      placeholder: 'Írja be',
      submit: 'Elküldés',
      virtual: 'Billentyűzet',
      readClipboard: 'Olvasás a vágólapról',
      clipboardPermissionDenied:
        'A vágólap engedélye megtagadva. Kérjük, engedélyezze a vágólaphoz való hozzáférést a böngészőjében.',
      clipboardReadError: 'Nem sikerült beolvasni a vágólapot',
      dropdownEnglish: 'angol',
      dropdownGerman: 'német',
      dropdownFrench: 'francia',
      dropdownRussian: 'orosz',
      shortcut: {
        title: 'Parancsikonok',
        custom: 'Egyedi',
        capture: 'Kattintson ide a parancsikon rögzítéséhez',
        clear: 'Tiszta',
        save: 'Mentés',
        captureTips:
          'A rendszerszintű billentyűk (például a Windows billentyű) rögzítéséhez teljes képernyős engedély szükséges.',
        enterFullScreen: 'Teljes képernyős mód váltása.'
      },
      leaderKey: {
        title: 'Leader billentyű',
        desc: 'Kerülje ki a böngésző korlátozásait, és küldje el a rendszer parancsikonjait közvetlenül a távoli gazdagépnek.',
        howToUse: 'Használat',
        simultaneous: {
          title: 'Egyidejű üzemmód',
          desc1: 'Tartsa lenyomva a Leader billentyűt, majd nyomja meg a gyorsbillentyűt.',
          desc2: 'Intuitív, de ütközhet a rendszer parancsikonjaival.'
        },
        sequential: {
          title: 'Szekvenciális mód',
          desc1:
            'Nyomja meg a Leader billentyűt → nyomja meg sorban a gyorsbillentyűt → nyomja meg újra a Leader billentyűt.',
          desc2: 'Több lépést igényel, de teljesen elkerülhető a rendszerütközések.'
        },
        enable: 'Leader billentyű engedélyezése',
        tip: 'Leader billentyűként beállítva ez a billentyű kizárólag gyorsbillentyű-indítóként működik, és elveszíti alapértelmezett viselkedését.',
        placeholder: 'Nyomja meg a Leader billentyűt',
        shiftRight: 'Jobb Shift',
        ctrlRight: 'Jobb Ctrl',
        metaRight: 'Jobb Win',
        submit: 'Elküldés',
        recorder: {
          rec: 'REC',
          activate: 'Billentyűk aktiválása',
          input: 'Kérjük, nyomja meg a parancsikont...'
        }
      }
    },
    mouse: {
      title: 'Egér',
      cursor: 'Kurzorstílus',
      default: 'Alapértelmezett kurzor',
      pointer: 'Mutató kurzor',
      cell: 'Cella kurzor',
      text: 'Szöveg kurzor',
      grab: 'Markoló kurzor',
      hide: 'Kurzor elrejtése',
      mode: 'Egér mód',
      absolute: 'Abszolút mód',
      relative: 'Relatív mód',
      direction: 'Görgő iránya',
      scrollUp: 'Görgessen felfelé',
      scrollDown: 'Görgessen le',
      speed: 'Görgő sebessége',
      fast: 'Gyors',
      slow: 'Lassú',
      requestPointer:
        'Relatív mód használata. Kattintson az asztalra, hogy megjelenjen az egérmutató.',
      resetHid: 'HID alaphelyzetbe állítása',
      hidOnly: {
        title: 'Csak HID mód',
        desc: 'Ha az egér és a billentyűzet nem válaszol, és az HID alaphelyzetbe állítása nem segít, akkor az NanoKVM és az eszköz közötti kompatibilitási probléma lehet. Próbálja engedélyezni az HID-Csak módot a jobb kompatibilitás érdekében.',
        tip1: 'Az HID-Csak mód engedélyezése leválasztja a virtuális U-lemezt és a virtuális hálózatot',
        tip2: 'HID-Csak módban a képrögzítés le van tiltva',
        tip3: 'A NanoKVM automatikusan újraindul az üzemmódváltás után',
        enable: 'Engedélyezze a HID-Csak módot',
        disable: 'A HID-Csak mód letiltása'
      }
    },
    image: {
      title: 'Képek',
      loading: 'Betöltés...',
      empty: 'Nem található semmi',
      mountMode: 'Felszerelési mód',
      mountFailed: 'Csatlakoztatás sikertelen',
      mountDesc:
        'Egyes rendszerekben szükséges lehet a virtuális lemez eltávolítása a távoli gépen, mielőtt a képet csatlakoztatja.',
      unmountFailed: 'A leválasztás nem sikerült',
      unmountDesc:
        'Egyes rendszereken manuálisan kell kiadnia a távoli gazdagépről a kép leválasztása előtt.',
      refresh: 'Frissítse a képlistát',
      attention: 'Figyelem',
      deleteConfirm: 'Biztosan törli ezt a képet?',
      okBtn: 'Igen',
      cancelBtn: 'Nem',
      tips: {
        title: 'Hogyan tölts fel képeket',
        usb1: 'Csatlakoztassa a NanoKVM-t a számítógépéhez USB-n keresztül.',
        usb2: 'Győződjön meg róla, hogy a virtuális lemez csatlakoztatva van (Beállítások - Virtuális lemez).',
        usb3: 'Nyissa meg a virtuális lemezt a számítógépén, és másolja a kép fájlt a virtuális lemez gyökérkönyvtárába.',
        scp1: 'Győződjön meg róla, hogy a NanoKVM és a számítógépe ugyanazon a helyi hálózaton van.',
        scp2: 'Nyisson meg egy terminált a számítógépén, és használja az SCP parancsot a kép fájl feltöltésére a /data könyvtárba a NanoKVM-en.',
        scp3: 'Példa: scp your-image-path root@your-nanokvm-ip:/data',
        tfCard: 'TF Kártya',
        tf1: 'Ez a módszer támogatott Linux rendszeren',
        tf2: 'Vegye ki a TF kártyát a NanoKVM-ből (a TELJES verzióhoz, először szedje szét a házat).',
        tf3: 'Helyezze a TF kártyát egy kártyaolvasóba, és csatlakoztassa a számítógépéhez.',
        tf4: 'Másolja a képfájlt a TF kártya /data könyvtárába.',
        tf5: 'Helyezze vissza a TF kártyát a NanoKVM-be.'
      }
    },
    script: {
      title: 'Szkriptek',
      upload: 'Feltöltés',
      run: 'Futtatás',
      runBackground: 'Háttérben futtatás',
      runFailed: 'Futtatás sikertelen',
      attention: 'Figyelem',
      delDesc: 'Biztosan törli ezt a fájlt?',
      confirm: 'Igen',
      cancel: 'Nem',
      delete: 'Törlés',
      close: 'Bezárás'
    },
    terminal: {
      title: 'Terminál',
      nanokvm: 'NanoKVM Terminál',
      serial: 'Soros port terminál',
      serialPort: 'Soros port',
      serialPortPlaceholder: 'Adja meg a soros portot',
      baudrate: 'Baudráta',
      parity: 'Paritás',
      parityNone: 'Nincs',
      parityEven: 'Páros',
      parityOdd: 'Páratlan',
      flowControl: 'Áramlásszabályozás',
      flowControlNone: 'Nincs',
      flowControlSoft: 'Szoftveres',
      flowControlHard: 'Hardveres',
      dataBits: 'Adatbitek',
      stopBits: 'Stop bitek',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Parancs küldése...',
      sent: 'Parancs elküldve',
      input: 'Adja meg a MAC címet',
      ok: 'Ok'
    },
    download: {
      title: 'Képletöltő',
      input: 'Adjon meg egy távoli képet URL',
      ok: 'Ok',
      disabled: '/data partíció RO, ezért nem tudjuk letölteni a képet',
      uploadbox: 'Dobja ide a fájlt, vagy kattintson a kiválasztáshoz',
      inputfile: 'Kérjük, írja be a képfájlt',
      NoISO: 'Nincs ISO',
      sha256: 'SHA-256 (opcionális)',
      sha256Placeholder: 'Adjon meg egy 64 karakteres SHA-256 ellenőrzőösszeget',
      invalidSHA256: 'A SHA-256 értékének 64 karakteres hexadecimális karakterláncnak kell lennie',
      failed: 'Sikertelen letöltés',
      success: 'Sikeres letöltés',
      checksumFailed: 'Sikertelen letöltés: a SHA-256 ellenőrzése sikertelen',
      cancel: 'Mégse',
      cancelFailed: 'A letöltés megszakítása sikertelen'
    },
    power: {
      title: 'Bekapcsolás',
      showConfirm: 'Megerősítés',
      showConfirmTip: 'Az áramellátási műveletekhez külön megerősítés szükséges',
      reset: 'Újraindítás',
      power: 'Bekapcsolás',
      powerShort: 'Bekapcsolás (rövid kattintás)',
      powerLong: 'Bekapcsolás (hosszú kattintás)',
      resetConfirm: 'Folytatja a visszaállítási műveletet?',
      powerConfirm: 'Folytatja az áramellátást?',
      okBtn: 'Igen',
      cancelBtn: 'Nem'
    },
    devices: {
      title: 'Eszközök',
      stale: 'Az eszközök élő állapota nem érhető el. Újracsatlakozás folyamatban.',
      empty: 'Nincs beállítva kamera- vagy mikrofonhely.',
      available: 'Elérhető',
      waiting: 'A gazdagép forrásra vár',
      hostOpen: 'A gazdagép nyitva',
      hostIdle: 'A gazdagép tétlen',
      sending: 'Küldés erről a böngészőről',
      black: 'Fekete kép',
      silence: 'Digitális csend',
      resuming: 'Folytatásra vár',
      stop: 'Megosztás leállítása',
      disconnect: 'Lecsatlakoztatás',
      takeover: 'Átvétel',
      refused: 'Használatban: {{owner}}, forrás: {{source}}',
      connectedSources_one: '{{count}} csatlakoztatott forrás',
      connectedSources_other: '{{count}} csatlakoztatott forrás',
      connectedSources: '{{count}} csatlakoztatott forrás',
      connection: {
        connecting: 'Csatlakozás',
        connected: 'Élő',
        disconnected: 'Újracsatlakozás'
      },
      share: {
        camera: 'Kamera megosztása',
        microphone: 'Mikrofon megosztása',
        usbDevice: 'USB megosztása'
      },
      permission: {
        denied: 'Letiltva a böngésző webhelybeállításaiban',
        prompt: 'A böngésző engedélyt fog kérni'
      },
      mic: {
        mute: 'Némítás',
        unmute: 'Némítás feloldása'
      },
      revoked: {
        released: 'A megosztás leállt',
        lease_expired: 'A bérlet lejárt, mielőtt ez a böngésző visszatért volna',
        admin_disconnect: 'Egy rendszergazda minden forrást lecsatlakoztatott',
        slot_removed: 'A helyet eltávolították',
        slot_changed: 'A helyet újrakonfigurálták',
        taken_over: 'Egy rendszergazda átvette ezt a helyet'
      },
      usb: {
        surrendered: 'Az USB-passthrough tartja a billentyűzetet és az egeret',
        surrenderedDesc:
          'A távoli gazdagép az importált eszközt látja a NanoKVM billentyűzete, egere és virtuális adathordozói helyett. Ezek a munkamenet leállásakor visszatérnek.',
        unsupported: 'A WebUSB Chromium-alapú böngészőt igényel',
        insecure: 'Ez az oldal nem HTTPS-en érkezik, ezért a böngésző visszatartja a WebUSB-t. Kapcsold be a HTTPS-t a Beállítások, Hálózat alatt.',
        session: '{{device}} átadása ({{mode}})',
        idle: 'Nincs passthrough munkamenet',
        mode: {
          hybrid: 'hibrid',
          exact: 'pontos'
        }
      }
    },
    settings: {
      title: 'Beállítások',
      display: {
        title: 'Kijelző',
        loading: 'Betöltés...',
        active: 'Aktív EDID',
        activeUnknown:
          'A NanoKVM indulás óta nem írt EDID-et, ezért nem tudni, milyen monitort lát a gazdagép.',
        appliedAt: 'Alkalmazva: {{time}}',
        download: 'Letöltés',
        downloadBackup: 'Előző letöltése',
        preset: 'Monitor-előbeállítás',
        presetPlaceholder: 'Válasszon monitort',
        upload: 'Feltöltés',
        selected: 'Kiválasztott EDID',
        errors: 'Hibák',
        warnings: 'Figyelmeztetések',
        info: 'Információ',
        unknownMonitor: 'Ismeretlen monitor',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Hang',
        audioNo: 'Nincs hang',
        extensionBlocks: 'Kiterjesztési blokkok: {{blocks}}',
        apply: 'Alkalmaz',
        applyTitle: 'Alkalmazza ezt az EDID-et?',
        before: 'Jelenlegi',
        after: 'Új',
        hdmiNotice: 'A képrögzítés az EDID írása alatt leáll, majd magától újraindul.',
        powerCycleNotice:
          'Az új EDID csak azután lép életbe, hogy az eszközt fizikailag áramtalanítja, majd újra bedugja.',
        powerCycleUnverified:
          'Az írás nem lett ellenőrizve, így a videochip megtartja azt, ami most benne van, amíg az eszközt fizikailag ki nem húzza a tápból, majd vissza nem dugja.',
        applied: 'Az EDID alkalmazva és ellenőrizve.',
        applyFailed: 'Az EDID alkalmazása nem sikerült.',
        busy: 'A videochip foglalt volt. Próbálja újra.',
        unsupported: 'Ez az eszköz nem támogatja az EDID módosítását.',
        toolMissing: 'Ebből a firmware-ből hiányzik az EDID eszköz.',
        noAudio: 'Ez az EDID nem jelez hangot, ezért a gazdagép leállíthatja a hangküldést.',
        oldVersion: 'Ez az EDID 1.4-nél régebbi verziót használ.',
        interlaced: 'A preferált időzítés váltottsoros.',
        tooLarge:
          'A preferált időzítés nagyobb, mint 1920x1080 60 Hz-en, amit a NanoKVM már nem tud rögzíteni.',
        recovery: 'Helyreállítás',
        recoveryNeeded:
          'A legutóbbi írás nem lett ellenőrizve, így a videochip EDID-területe ismeretlen állapotban van. Állítsa vissza a gyári EDID-et, hogy az állapot ismét ismert legyen.',
        recoveryDesc:
          'Írjon vissza egy ismert EDID-et a videochipre, ha az alkalmazott EDID kép nélkül hagyta a gazdagépet.',
        restoreFactory: 'Gyári EDID visszaállítása',
        restoreBackup: 'Előző EDID visszaállítása',
        restoreTitle: 'Visszaállítja ezt az EDID-et?',
        restoreFactoryTarget: 'A NanoKVM-mel szállított gyári EDID.',
        restoreBackupTarget: 'A legutóbbi mentés, alkalmazva: {{time}}.',
        restoreNotice:
          'A visszaállítás ugyanúgy íródik, mint az alkalmazás, és ugyanazokkal a következményekkel jár.',
        restored: 'Az EDID visszaállítva és ellenőrizve.',
        restoreFailed: 'Az EDID visszaállítása nem sikerült.',
        mismatchTitle: 'Kiírt és visszaolvasott adat',
        mismatchWritten: 'Kiírt',
        mismatchRead: 'Visszaolvasott',
        restoreOkBtn: 'Visszaállítás',
        hardware: 'Észlelt hardver: {{hardware}}',
        hardwareUnknown: 'Ismeretlen',
        confirmWord: 'ALKALMAZ',
        confirmPrompt: 'Írja be: {{word}}, hogy az alkalmazás gomb aktívvá váljon.',
        okBtn: 'Alkalmaz',
        cancelBtn: 'Mégse'
      },
      presentation: {
        title: 'USB-megjelenés',
        loading: 'Betöltés...',
        current: 'Jelenlegi USB-megjelenés',
        noProfile: 'Nincs alkalmazott profil',
        linked: 'Összekapcsolt funkciók',
        hostState: 'A gazdagép USB-je',
        hostUnbound: 'A vezérlő nincs hozzákötve',
        hdmiState: 'HDMI-bemenet',
        hdmiSignal: 'Van jel',
        hdmiUnreported: 'Még nincs jelentés a rögzítésről',
        endpoints: 'Endpointok',
        fifos: 'FIFO-helyek',
        pending: 'Függőben lévő változtatások',
        pendingEdits: 'Nem mentett identitásmódosítások',
        pendingProfile: '{{profile}} ki van választva, de nincs alkalmazva',
        pendingNone: 'Nincs',
        lastApply: 'Utolsó alkalmazás',
        applyFailed: 'Sikertelen: {{profile}}, ekkor: {{time}}',
        applyClean: 'Nincs rögzített hiba',
        lastKnownGood: 'Utolsó ismert jó állapot',
        rollbackTarget: 'Visszaállítás célja',
        rollbackNone: 'Nincs',
        powerCyclePending:
          'A vezérlőt elvették a gazdagéptől. Kapcsolja ki, majd be a csatlakoztatott számítógépet, hogy az eszköz visszatérjen.',
        rollback: 'Visszaállítás',
        rollbackTitle: 'Visszaáll erre: {{profile}}?',
        rollbackDesc: 'A gadget újra felsorolódik; az USB-funkciók rövid időre kiesnek.',
        profile: 'USB-profil',
        builtIn: 'beépített',
        descriptors: 'leírók',
        imported: 'importált',
        clone: 'Klónozás',
        cloneTitle: 'A profil klónozása',
        cloneToEdit:
          'A beépített profilok csak olvashatók maradnak. Klónozza ezt a profilt az identitásának szerkesztéséhez.',
        profileName: 'Profil neve',
        profileNameHint: 'Kisbetűk, számjegyek, pontok, aláhúzások és kötőjelek.',
        import: 'Csomag importálása',
        export: 'Csomag exportálása',
        delete: 'Törlés',
        deleteTitle: 'Törli ezt a profilt?',
        deleteDesc: '{{profile}} törlése a NanoKVM-ről.',
        identity: 'USB-identitás',
        preset: 'Előbeállított identitás',
        presetPlaceholder: 'Identitás átvétele egy ismert eszközről',
        presetHint:
          'Egy előbeállítás kitölti a Vendor ID-t, a Product ID-t és a két névmezőt. Leírókat nem hoz magával.',
        presetSource: 'Az identitás úgy, ahogy a(z) {{source}} rögzíti',
        vendorId: 'Vendor ID',
        foreignVendor: 'Ez a Vendor ID egy másik gyártóé',
        productId: 'Product ID',
        bcdUSB: 'USB-verzió',
        bcdDevice: 'Eszközverzió',
        manufacturer: 'Gyártó',
        product: 'Termék',
        serial: 'Sorozatszám',
        configuration: 'Konfigurációs karakterlánc',
        hidLayout: 'HID-eszközök',
        hidRoleKeyboard: 'Billentyűzet',
        hidRoleRelative: 'Egér (relatív)',
        hidRoleAbsolute: 'Mutató (abszolút)',
        hidOff: 'Nincs jelen',
        hidInterface: '{{index}}. interfész',
        hidBootKeyboardShared:
          'A billentyűzet osztozik egy interfészen, ezért már nem kínál boot protokollú jelentést. Néhány BIOS és UEFI nem fogja látni.',
        functions: 'Funkciók',
        descriptorAssets: 'Tárolt leírófájlok: {{count}}',
        endpointUse:
          'IN {{inUse}} használatban, {{inFree}} szabad; OUT {{outUse}} használatban, {{outFree}} szabad',
        preview: 'Ellenőrzés',
        save: 'Mentés',
        apply: 'Alkalmaz',
        applyTitle: 'Alkalmazza ezt az USB-profilt?',
        applyDesc:
          'A NanoKVM a(z) {{profile}} profilt fogja mutatni a csatlakoztatott számítógépnek.',
        reconnect:
          'A billentyűzet, az egér és a többi USB-funkció rövid időre lecsatlakozik, amíg a gadget újra hozzákötődik.',
        applyLinks: 'Összekapcsolja: {{functions}}',
        applyRemoves: 'Eltávolítja: {{functions}}',
        applyNoHid:
          'Ezután az alkalmazás után nem marad HID-funkció. A billentyűzet és az egér működése megszűnik.',
        applyRollback: 'A sikertelen alkalmazás visszatér ehhez: {{profile}}.',
        recoveryPowerCycle:
          'Ezt az alkalmazást egyetlen HID sem éli túl, így a válaszolást abbahagyó gazdagép csak ki- és bekapcsolással állítható helyre.',
        recoveryReboot:
          'Egy interfész eltűnik az összetett eszközből; a gazdagépnek újraindításra lehet szüksége, hogy a többit újra hozzákösse.',
        recoveryHdmiReset:
          'Egy videofunkció újraépül, ezért a mögötte lévő rögzítési lánc alaphelyzetbe áll.',
        recoveryReconnect:
          'A gazdagép újra felsorolja az eszközt; az USB-funkciók rövid időre kiesnek.',
        cancel: 'Mégse',
        noFunctions: 'Nincs összekapcsolt funkció',
        loadFailed: 'A megjelenési profilok betöltése nem sikerült',
        operationFailed: 'Az USB-megjelenés művelete nem sikerült'
      },
      passthrough: {
        title: 'USB-átjátszás',
        loading: 'Betöltés...',
        mode: 'Mód',
        hybrid: 'Hibrid',
        exact: 'Pontos',
        hybridDesc:
          'Megtartja a boot billentyűzetet és a relatív egeret, kompatibilis eszközökhöz.',
        exactDesc: 'A NanoKVM minden USB-funkcióját az átjátszott eszközre cseréli.',
        hybridWarning: 'A hibrid mód elérhetően hagyja a billentyűzetet és a relatív egeret',
        hybridWarningDesc:
          'A tároló, az USB-hálózat és az abszolút mutató lecsatlakozik, amíg az átjátszott funkció aktív.',
        hidWarning:
          'Az átjátszás indítása átadja a billentyűzetet, az egeret és a virtuális adathordozókat',
        hidWarningDesc:
          'A NanoKVM-nek egyetlen USB-eszközvezérlője van, és a proxynak az egész kell. Amíg fut egy munkamenet, a távoli gép ezért az átjátszott eszközt látja a NanoKVM billentyűzete, egere és virtuális adathordozói helyett. Ezek maguktól visszatérnek, amint a munkamenet leáll. Ez a webes felület ettől függetlenül működik, így a munkamenetet bármikor leállíthatja erről az oldalról.',
        hidWarningSafeDesc:
          'A NanoKVM-nek egyetlen USB-eszközvezérlője van, és a proxynak az egész kell. Amíg fut egy munkamenet, a távoli gép ezért az átjátszott eszközt látja a NanoKVM billentyűzete, egere és virtuális adathordozói helyett. Ezek visszatérnek, amint a munkamenet leáll.',
        isoLabel: 'Izokron átvitelek engedélyezése',
        isoHint:
          'Beengedi a webkamerákat, mikrofonokat és más folyamatos eszközöket. Senki sem mérte meg, mit bír ez a hardver.',
        isoWarning:
          'Az izokron átvitel itt kipróbálatlan, és a munkamenet leállításáig foghatja a billentyűzetet és az egeret',
        info: {
          title: 'Tudnivalók',
          hybrid:
            'A hibrid mód elérhetően hagyja a billentyűzetet és a relatív egeret. A tároló, az USB-hálózat és az abszolút mutató lecsatlakozik, amíg az átjátszott eszköz aktív.',
          exact:
            'A pontos mód a NanoKVM minden USB-funkcióját az átjátszott eszközre cseréli. A billentyűzet, az egér és a virtuális adathordozók maguktól visszatérnek, amint a munkamenet leáll.',
          udc: 'A NanoKVM-nek egyetlen USB-eszközvezérlője van, és a proxynak az egész kell — ezért tűnnek el a fenti funkciók a munkamenet teljes idejére.',
          web: 'Ez a webes felület ettől függetlenül működik, így a munkamenetet bármikor leállíthatja erről az oldalról.',
          network:
            'Az átjátszást Etherneten vagy Wi-Fin indítsa. A NanoKVM USB-hálózatáról való indítást elutasítja a rendszer, mert az a kapcsolat eltűnne.',
          iso: 'A webkamerákat, mikrofonokat és más izokron eszközöket a rendszer elutasítja, amíg nem engedélyezi az izokron átvitelt. Ez az út működik, de ezen a hardveren soha nem mérték meg, ezért tekintse az átviteli sebességét ismeretlennek.',
          camera:
            'A böngésző kamerája és mikrofonja az Eszközök alatt továbbra is a bevált módja annak, hogy a távoli gép kapjon egyet.'
        },
        session: 'Munkamenet',
        activeDesc: 'Egy eszköz be van importálva, és a proxy tartja az USB-vezérlőt.',
        inactiveDesc:
          'Nincs futó munkamenet. A billentyűzet, az egér és a virtuális adathordozók rendesen működnek.',
        device: 'Eszköz',
        busId: 'Busz-azonosító',
        speed: 'Sebesség',
        exporter: 'Exportáló',
        local: 'Importálva mint',
        localValue: '{{bus}}. busz, {{address}} cím',
        udc: 'USB-vezérlő',
        pid: 'Proxy PID',
        startedAt: 'Indítva',
        isoDevice:
          'Ez az eszköz izokron végpontokon sugároz, amit ezen a hardveren még soha nem mértek',
        exporterLabel: 'Az exportáló címe',
        exporterHint:
          'A gép és a port, amelyet a NanoKVM felhív. Az alábbi alagúton át ez {{exporter}}.',
        busIdLabel: 'Busz-azonosító a saját gépén',
        busIdHint: 'Az a busid, amelyet az usbip list -l kiír az eszközhöz, például {{example}}.',
        start: 'Átjátszás indítása',
        stop: 'Átjátszás leállítása',
        startTitle: 'Elindítja az USB-átjátszást?',
        startDevice: 'A NanoKVM importálni fogja a következőt: {{busId}} innen: {{exporter}}.',
        startHid:
          'Az USB-billentyűzet, az egér és a virtuális adathordozók a munkamenet teljes ideje alatt nem működnek, és maguktól újraindulnak, amint leállítja.',
        startIso:
          'A webkamerákhoz és más izokron eszközökhöz indítás előtt be kell kapcsolni az izokron kapcsolót.',
        startWeb:
          'Ez a webes felület tovább működik, így a munkamenetet bármikor leállíthatja erről az oldalról.',
        startNetwork:
          'Ezt az oldalt Etherneten vagy Wi-Fin használja. A NanoKVM USB-hálózatáról való indítást elutasítja a rendszer, mert az a kapcsolat eltűnne.',
        okBtn: 'Indítás',
        cancelBtn: 'Mégse',
        instructions: 'A saját gépén',
        instructionsDesc:
          'Szándékosan nincs telepítendő kliensügynök. Futtassa ezeket a szokásos usbip parancsokat azon a gépen, amelyhez az eszköz csatlakozik.',
        copyFailed: 'A másolás nem sikerült. Másolja a parancsot kézzel.',
        copyInsecure: 'Ez az oldal nem HTTPS-en érkezik, ezért a böngésző letiltotta a másolást. Másold a parancsot kézzel, vagy kapcsold be a HTTPS-t a Beállítások, Hálózat alatt.',
        directNote:
          'Alagút nélkül az usbipd-nek elérhetőnek kell lennie a hálózatán, és a fenti exportálócímnek rá kell mutatnia. Az usbip titkosítatlanul viszi az eszközt, ezért az alagút az ajánlott.',
        steps: {
          modprobe: {
            title: 'Töltse be az exportáló oldali illesztőprogramot',
            desc: 'Az usbip-host teszi lehetővé, hogy a kernel átadjon egy helyi eszközt. Alapból nincs betöltve.'
          },
          list: {
            title: 'Keresse meg az eszközt',
            desc: 'Kiírja az összes helyi eszközt a busid-jével és a gyártó:termék párossal. Jegyezze fel a kívánt eszköz busid-jét.'
          },
          bind: {
            title: 'Kösse hozzá az usbip-hez',
            desc: 'Elveszi az eszközt a szokásos illesztőprogramjától, így az a leválasztásig nem működik ezen a gépen.'
          },
          serve: {
            title: 'Tegye elérhetővé',
            desc: 'Az usbipd az előtérben marad, és megvárja, amíg a NanoKVM importálja az eszközt.',
            notice:
              'A szokásos usbipd-nek nincs figyelőcím-kapcsolója, minden felületen figyel. Tartsa a(z) {{port}} portot zárva a tűzfalán, és csak az alábbi alagutat engedje hozzá.'
          },
          tunnel: {
            title: 'Irányítsa a NanoKVM-re',
            desc: 'Fordított SSH-alagút: a NanoKVM saját visszacsatolási címén a(z) {{port}} port lesz ezen a gépen az usbipd. Hagyja futni a teljes munkamenet alatt.'
          },
          exporter: {
            title: 'Ezt adja meg exportálóként',
            desc: 'Írja be ezt a fenti exportáló mezőbe, adja meg a busz-azonosítót, majd indítsa el a munkamenetet.'
          },
          unbind: {
            title: 'Adja vissza az eszközt',
            desc: 'A munkamenet leállása után ez visszaadja az eszközt a szokásos illesztőprogramjának ezen a gépen.'
          }
        }
      },
      mcp: {
        title: 'MCP-szolgáltatás',
        service: 'MCP távoli vezérlés',
        serviceDesc:
          'Megbízható MCP-kliensek számára a billentyűzet és az egér vezérlésének, valamint képernyőképek készítésének engedélyezése',
        securityWarning:
          'Az API-kulcs birtokában bárki vezérelheti a távoli gazdagépet és láthatja annak képernyőjét. Használjon HTTPS-t, és csak megbízható hálózatokon engedélyezze.',
        endpoint: 'Végpont',
        apiKey: 'API-kulcs',
        regenerateConfirmTitle: 'Újragenerálja az MCP API-kulcsot?',
        regenerateConfirmDesc: 'A jelenlegi kulcs azonnal érvényét veszti.',
        enableConfirmTitle: 'Engedélyezi a külső MCP-vezérlést?',
        enableConfirmDesc:
          'Az MCP engedélyezése leállítja a PicoClaw-t, és bezár minden aktív PicoClaw-munkamenetet.',
        failed: 'Az MCP-művelet sikertelen',
        copyFailed: 'A másolás sikertelen. Másolja kézzel.',
        copyInsecure: 'Ez az oldal nem HTTPS-en érkezik, ezért a böngésző letiltotta a másolást. Másold kézzel, vagy kapcsold be a HTTPS-t a Beállítások, Hálózat alatt.',
        okBtn: 'Megerősítés',
        cancelBtn: 'Mégse'
      },
      about: {
        title: 'NanoKVM Névjegy',
        information: 'Információ',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Alkalmazás verzió',
        applicationTip: 'NanoKVM webalkalmazás verziója',
        image: 'Képfájl verzió',
        imageTip: 'NanoKVM rendszerkép verziója',
        deviceKey: 'Eszköz kulcs',
        community: 'Közösség',
        hostname: 'Gazdanév',
        hostnameUpdated: 'Gazdanév frissítve. Az alkalmazáshoz indítsa újra.',
        ipType: {
          Wired: 'Vezetékes',
          Wireless: 'Vezeték nélküli',
          Other: 'Egyéb'
        }
      },
      appearance: {
        title: 'Megjelenés',
        display: 'Kijelző',
        language: 'Nyelv',
        languageDesc: 'Válassza ki a felület nyelvét',
        webTitle: 'Webcím',
        webTitleDesc: 'A weboldal címének testreszabása',
        menuBar: {
          title: 'Menüsor',
          mode: 'Megjelenítési mód',
          modeDesc: 'Menüsor megjelenítése a képernyőn',
          modeOff: 'Ki',
          modeAuto: 'Automatikus elrejtés',
          modeAlways: 'Mindig látható',
          keyboardLedStatus: 'Billentyűzár-jelzők',
          keyboardLedStatusDesc:
            'A távoli számítógép Num Lock, Caps Lock és Scroll Lock állapotának megjelenítése',
          icons: 'Almenü ikonok',
          iconsDesc: 'Almenüikonok megjelenítése a menüsorban'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Távoli billentyűzárak állapota',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Be',
        off: 'Ki',
        unknown: 'Ismeretlen'
      },
      device: {
        title: 'Eszköz',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Soha',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 óra'
        },
        ssh: {
          description: 'Engedélyezze a SSH távoli hozzáférést',
          tip: 'Az engedélyezés előtt állítson be erős jelszót (Fiók - Jelszó módosítása)'
        },
        advanced: 'Speciális beállítások',
        swap: {
          title: 'Csere',
          disable: 'Letiltás',
          description: 'Állítsa be a swap fájl méretét',
          tip: 'Ennek a funkciónak az engedélyezése lerövidítheti az SD-kártya élettartamát!'
        },
        mouseJiggler: {
          title: 'Mouse Jiggler',
          description: 'A távoli gazdagép alvó állapotának megakadályozása',
          disable: 'Letiltás',
          absolute: 'Abszolút mód',
          relative: 'Relatív mód'
        },
        mdns: {
          description: 'Engedélyezze az mDNS felderítési szolgáltatást',
          tip: 'Kikapcsolás, ha nincs rá szükség'
        },
        hdmi: {
          description: 'HDMI/monitor kimenet engedélyezése',
          idleTimeoutTitle: 'Inaktív rögzítés időkorlátja',
          idleTimeoutDescription: 'A HDMI-rögzítés leállítása, ha nincs aktív néző ennyi ideig:',
          minutes: 'perc'
        },
        autostart: {
          title: 'Automatikus indítási parancsfájlok beállításai',
          description: 'A rendszer indításakor automatikusan futó szkriptek kezelése',
          new: 'Új',
          deleteConfirm: 'Biztosan törli ezt a fájlt?',
          yes: 'Igen',
          no: 'Nem',
          scriptName: 'Automatikusan induló szkript neve',
          scriptContent: 'A szkripttartalom automatikus indítása',
          settings: 'Beállítások'
        },
        hidOnly: 'HID-Csak mód',
        hidOnlyDesc:
          'A virtuális eszközök emulálásának leállítása, csak az alapvető HID vezérlés megtartásával',
        disk: 'Virtuális lemez',
        diskDesc: 'Mount virtual U-disk on the remote host',
        network: 'Virtuális hálózat',
        networkDesc: 'Virtuális hálózati kártya csatlakoztatása a távoli gazdagépen',
        networkProtocol: 'Hálózati protokoll',
        networkProtocolDesc: 'NCM modern gazdagépekhez, RNDIS régebbi Windowshoz',
        media: {
          title: 'Kamera- és mikrofonhelyek',
          desc: 'Adja meg, mely médiaeszközöket tölthetik ki a böngészők. A végpontkeretet a rendszer az USB-profil alkalmazásakor ellenőrzi. A mentés újraszámlálja az eszközt, és lecsatlakoztatja a csatlakozott böngészőket.',
          cameras: 'Kamerák',
          microphones: 'Mikrofonok',
          name: 'Név',
          namePlaceholder: 'A célgépen jelenik meg',
          addCamera: 'Kamera hozzáadása',
          addMicrophone: 'Mikrofon hozzáadása',
          remove: 'Eltávolítás',
          cameraDefault: 'NanoKVM kamera {{index}}',
          microphoneDefault: 'NanoKVM mikrofon {{index}}',
          nameRequired: 'Minden helyhez név kell.',
          unsupported:
            'Ez a kernel nem tudja elnevezni a médiaeszközöket, ezért a gépek az alapértelmezett nevet mutatják.',
          save: 'Helyek mentése',
          disconnect: 'Lecsatlakoztatás',
          disconnectAll: 'Minden forrás lecsatlakoztatása',
          limit: 'A kamera- és mikrofonhelyek összesen legfeljebb nyolcan lehetnek.',
          failed: 'A médiahelyeket nem sikerült frissíteni.'
        },
        reboot: 'Újraindítás',
        rebootDesc: 'Biztos, hogy újra akarja indítani a NanoKVM-t?',
        okBtn: 'Igen',
        cancelBtn: 'Nem'
      },
      network: {
        title: 'Hálózat',
        wifi: {
          title: 'Wi-Fi',
          description: 'Wi-Fi beállítása',
          apMode: 'Az AP mód engedélyezve van, csatlakozzon a Wi-Fihez a QR-kód beolvasásával',
          connect: 'Wi-Fi csatlakoztatása',
          connectDesc1: 'Adja meg a hálózat SSID-jét és jelszavát',
          connectDesc2: 'Adja meg a jelszót a hálózathoz való csatlakozáshoz',
          disconnect: 'Biztosan bontja a hálózati kapcsolatot?',
          failed: 'A csatlakozás sikertelen, próbálja újra.',
          ssid: 'Név',
          password: 'Jelszó',
          joinBtn: 'Csatlakozás',
          confirmBtn: 'OK',
          cancelBtn: 'Mégse'
        },
        tls: {
          description: 'HTTPS protokoll engedélyezése',
          tip: 'Figyelem: A HTTPS használata növelheti a késleltetést, különösen MJPEG videó módban.'
        },
        bridge: {
          title: 'Hálózati híd',
          twoDevices:
            'A router a NanoKVM-et és a vezérelt számítógépet két külön eszközként látja, mindegyiket saját címmel.',
          loading: 'Betöltés...',
          state: 'Állapot',
          states: {
            disabled: 'Kikapcsolva',
            enabled: 'Bekapcsolva',
            rolledBack: 'Visszaállítva',
            failed: 'Sikertelen',
            pending: 'Folyamatban'
          },
          uplink: 'Uplink',
          ports: 'Portok',
          protocol: 'Eszközprotokoll',
          up: 'aktív',
          down: 'inaktív',
          noLink: 'nincs link',
          enableTitle: 'Bekapcsolja a hálózati hidat?',
          disableTitle: 'Kikapcsolja a hálózati hidat?',
          reconnect:
            'A cím áthelyezése közben a felügyeleti kapcsolat rövid időre megszakad, majd újra létrejön.',
          rollback: 'Ha az ellenőrzés sikertelen, a korábbi konfiguráció automatikusan visszaáll.',
          enableBtn: 'Bekapcsolás',
          disableBtn: 'Kikapcsolás',
          cancelBtn: 'Mégse',
          interrupted:
            'A kapcsolat megszakadt az alkalmazás közben. A jelenlegi állapot újraellenőrzése folyik.',
          pendingNotice: 'A híd módosítása még folyamatban van, vagy befejezés előtt megszakadt.',
          revert: 'Korábbi konfiguráció visszaállítása',
          rolledBackNotice:
            'A legutóbbi módosítás visszavonásra került, és a korábbi konfiguráció visszaállt.',
          verifyFailed: 'Az ellenőrzés sikertelen: {{gates}}',
          gates: {
            address: 'cím',
            gateway: 'átjáró',
            inbound: 'bejövő kapcsolat'
          },
          inboundWeak:
            'A bejövő ellenőrzés csak azért sikerült, mert a NanoKVM önmagához kapcsolódott. Ez azt bizonyítja, hogy a webszolgáltatás figyel és helyben elérhető, nem azt, hogy a hálózat felől érkező kérés eljut hozzá.',
          noCarrier:
            'Nincs link a(z) {{port}} porton. A hídnak nincs útja a hálózat felé, amíg nem csatlakozik kábel.',
          loop: 'A router a(z) {{port}} porton is tanulódik, tehát az a port egy második út ugyanahhoz a hálózathoz. A spanning tree ki van kapcsolva, így itt semmi nem bontja meg a hurkot: bontsa a két út egyikét.',
          failedNotice:
            'A legutóbbi módosítást nem sikerült visszavonni. A NanoKVM lehet, hogy csak a Wi-Fi AP-n vagy soros konzolon érhető el.'
        },
        dns: {
          title: 'DNS',
          description: 'DNS-kiszolgálók beállítása a NanoKVM számára',
          mode: 'Mód',
          dhcp: 'DHCP',
          manual: 'Kézi',
          add: 'DNS hozzáadása',
          save: 'Mentés',
          invalid: 'Adjon meg egy érvényes IP-címet',
          noDhcp: 'Jelenleg nincs elérhető DHCP DNS',
          saved: 'DNS-beállítások mentve',
          saveFailed: 'Nem sikerült menteni a DNS-beállításokat',
          unsaved: 'Nem mentett módosítások',
          maxServers: 'Legfeljebb {{count}} DNS-kiszolgáló engedélyezett',
          dnsServers: 'DNS-kiszolgálók',
          dhcpServersDescription: 'A DNS-kiszolgálók automatikusan DHCP-n keresztül érkeznek',
          manualServersDescription: 'A DNS-kiszolgálók kézzel szerkeszthetők',
          networkDetails: 'Hálózati részletek',
          interface: 'Interfész',
          ipAddress: 'IP-cím',
          subnetMask: 'Alhálózati maszk',
          router: 'Router',
          none: 'Nincs'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC-kiszolgáló',
        description:
          'Bármely VNC-kliens láthatja a távoli képernyőt, és használhatja a billentyűzetet és az egeret, a NanoKVM-fiókjával bejelentkezve',
        port: 'Port',
        portDescription: 'Csatlakozzon erre a portra a NanoKVM címén'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Memóriaoptimalizálás',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Memória csere',
          tip: 'Ha a memóriaoptimalizálás engedélyezése után is fennállnak a problémák, próbálja meg engedélyezni a swap memóriát. Ez alapértelmezés szerint a swap fájl méretét 256MB értékre állítja be, amely a "Beállítások > Eszköz" menüpontban állítható be.'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Betöltés...',
        notInstall: 'Tailscale nem található! Kérem, telepítse.',
        install: 'Telepítés',
        installing: 'Telepítés folyamatban',
        failed: 'Telepítés sikertelen',
        retry: 'Frissítse az oldalt, majd próbálja újra. Vagy próbálja meg manuálisan telepíteni.',
        download: 'Letöltés a',
        package: 'telepítési csomag',
        unzip: 'és kicsomagolás',
        upTailscale: 'Töltsön fel tailscale-t a NanoKVM /usr/bin/ könyvtárába',
        upTailscaled: 'Töltsön fel tailscaled-t a NanoKVM /usr/sbin/ könyvtárába',
        refresh: 'Frissítse az aktuális oldalt',
        notRunning: 'Tailscale nem fut. Kérjük, indítsa el a folytatáshoz.',
        run: 'Indítás',
        notLogin:
          'Az eszköz még nincs kötve. Kérem, jelentkezzen be és kösse az eszközt a fiókjához.',
        urlPeriod: 'Ez az url 10 percig érvényes',
        login: 'Bejelentkezés',
        loginSuccess: 'Sikeres bejelentkezés',
        enable: 'Tailscale engedélyezése',
        deviceName: 'Eszköz neve',
        deviceIP: 'Eszköz IP',
        account: 'Fiók',
        logout: 'Kijelentkezés',
        logoutDesc: 'Biztos, hogy ki szeretne jelentkezni?',
        uninstall: 'Eltávolítás Tailscale',
        uninstallDesc: 'Biztosan eltávolítja a Tailscale alkalmazást?',
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
        loading: 'Betöltés...',
        notInstall: 'Nincs telepítve',
        notConfigured: 'Nincs beállítva',
        stopped: 'Leállítva',
        running: 'Fut',
        connected: 'Csatlakozva',
        error: 'Hiba',
        atBoot: 'indul rendszerindításkor',
        notAtBoot: 'nem indul rendszerindításkor',
        arguments: 'Argumentumok',
        argumentsTip: 'A szolgáltatásnak indításkor átadott parancssori argumentumok.',
        env: 'Környezeti változók',
        envKey: 'Név',
        envValue: 'Érték',
        envAdd: 'Változó hozzáadása',
        envRemove: 'Eltávolítás',
        configured: 'Beállítva',
        save: 'Mentés',
        saved: 'A beállítások elmentve',
        start: 'Indítás',
        stop: 'Leállítás',
        restart: 'Újraindítás',
        logs: 'Naplók',
        logsEmpty: 'Még nincsenek naplóbejegyzések',
        refresh: 'Frissítés',
        binary: 'Bináris',
        binaryShipped: 'A firmware része',
        binaryCustom: 'Egyéni bináris',
        binaryUpload: 'Bináris feltöltése',
        binaryRevert: 'Gyári bináris visszaállítása',
        binaryRevertDesc:
          'Törli a feltöltött binárist, és visszaállítja a firmware-rel szállított változatot?',
        serverWarning: 'A korlátozás nélküli kiszolgáló nyílt proxyként működik',
        noHealthSignal:
          'Ez a szolgáltatás nem jelez állapotot, így a NanoKVM csak azt tudja, hogy a folyamat fut, azt nem, hogy az alagút kapcsolódott-e.',
        memoryWarning: 'Több távoli elérési szolgáltatás egyidejű futtatása kimerítheti a memóriát',
        resources: 'Erőforrások',
        memory: {
          title: 'Memóriakorlát',
          description:
            'A newt Go-heapjét {{limit}} MiB-ra korlátozza a következő újraindításától. Ez a saját korlátja, nem a Tailscale-é; kikapcsolva a Go alapértelmezése marad, a GOGC=50 pedig mindkét esetben érvényes.',
          noRuntime:
            'A wstunnel Rust nyelvű: nincs szemétgyűjtője és nincs beállítható heapkorlátja, a munkaszálai pedig már így is az eszköz egyetlen CPU-jához igazodnak.',
          notApplicable: 'Nem alkalmazható'
        },
        swap: {
          title: 'Lapozófájl',
          description:
            'Egy 256 MB-os lapozófájlt hoz létre az SD-kártyán. Rendszerszintű: ugyanaz a lapozófájl szolgálja ki a Tailscale-t, a KVM-kiszolgálót és mindent mást az eszközön.'
        },
        okBtn: 'Igen',
        cancelBtn: 'Nem'
      },
      update: {
        title: 'Frissítés keresése',
        queryFailed: 'Verzió lekérdezése sikertelen',
        updateFailed: 'Frissítés sikertelen. Kérem, próbálja újra.',
        isLatest: 'Ön már a legfrissebb verziót használja.',
        rebooting:
          'Az új kernel telepítése és újraindítás folyamatban. Ez néhány percig tarthat; ne kapcsolja ki az eszközt.',
        kernelUpdate:
          'Ez a frissítés a(z) {{version}} kernelt telepíti. Az eszköz újraindul, és magától visszatér a jelenlegi kernelhez, ha az új nem indul el.',
        rolledBack:
          'A(z) {{version}} kernel nem indult el, ezért az eszköz visszaállt az előző kernelre.',
        available: 'Frissítés elérhető. Biztos, hogy frissít?',
        updating: 'Frissítés elkezdődött. Kérem várjon...',
        confirm: 'Megerősítés',
        cancel: 'Mégse',
        preview: 'Frissítések előnézete',
        previewDesc: 'Korai hozzáférést kap az új funkciókhoz és fejlesztésekhez',
        previewTip:
          'Kérjük, vegye figyelembe, hogy az előzetes verziók hibákat vagy hiányos funkciókat tartalmazhatnak!',
        customServer: {
          title: 'Egyéni frissítési kiszolgáló',
          desc: 'Online frissítések keresése és letöltése a megadott kiszolgálóról',
          invalidUrl:
            'Adjon meg egy érvényes HTTP- vagy HTTPS-kiszolgálókönyvtárat lekérdezés, töredékazonosító és latest.json nélkül.',
          loadFailed: 'Nem sikerült betölteni a frissítési kiszolgáló beállításait.',
          saveFailed: 'Nem sikerült menteni a frissítési kiszolgáló beállításait.',
          saved: 'A frissítési kiszolgáló beállításai mentve.',
          save: 'Mentés',
          confirmTitle: 'Egyéni frissítési kiszolgálót használ?',
          confirmDesc:
            'Az SHA-512 csak azt ellenőrzi, hogy a csomag megfelel-e a kiszolgáló által biztosított jegyzéknek. Nem igazolja, hogy a csomag hivatalos NanoKVM-kiadás. Egy hibás vagy rosszindulatú kiszolgáló használhatatlanná teheti az eszközt, adatvesztést okozhat, vagy veszélyeztetheti a rendszert.',
          confirm: 'Használat mindenképpen',
          previewDisabled:
            'Az előzetes frissítések nem érhetők el, amíg egyéni frissítési kiszolgáló van engedélyezve.'
        },
        offline: {
          title: 'Offline frissítések',
          desc: 'Frissítés helyi telepítőcsomaggal',
          upload: 'Feltöltés',
          checksumPlaceholder: 'SHA-256 ellenőrzőösszeg (opcionális)',
          invalidChecksum: 'A SHA-256 ellenőrzőösszegnek 64 hexadecimális karakterből kell állnia.',
          checksumMismatch: 'Az SHA-256 ellenőrzése sikertelen. Lehet, hogy a csomag sérült.',
          invalidName: 'Érvénytelen fájlnévformátum. Kérjük, töltse le a GitHub kiadásaiból.',
          updateFailed: 'Frissítés sikertelen. Kérem, próbálja újra.'
        }
      },
      account: {
        title: 'Fiók',
        webAccount: 'Webes fiók neve',
        role: 'Szerepkör',
        roles: {
          admin: 'Rendszergazda',
          user: 'Felhasználó'
        },
        password: 'Jelszó',
        updateBtn: 'Update',
        logoutBtn: 'Kijelentkezés',
        logoutDesc: 'Biztos, hogy ki szeretne jelentkezni?',
        okBtn: 'Igen',
        cancelBtn: 'Nem',
        users: {
          title: 'Felhasználók',
          create: 'Felhasználó létrehozása',
          enabled: 'Engedélyezve',
          disabled: 'Letiltva',
          deviceOwner: 'Az eszköz tulajdonosa',
          resetPassword: 'Jelszó visszaállítása',
          delete: 'Törlés',
          deleteConfirm: 'Törli ezt a felhasználót és visszavonja az összes munkamenetét?',
          created: 'Felhasználó létrehozva',
          deleted: 'Felhasználó törölve',
          passwordUpdated: 'Jelszó frissítve',
          loadFailed: 'Nem sikerült betölteni a felhasználókat',
          saveFailed: 'Nem sikerült menteni a felhasználót',
          deleteFailed: 'Nem sikerült törölni a felhasználót'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Asszisztens',
      empty: 'Nyissa meg a panelt, és indítsa el a feladatot.',
      inputPlaceholder: 'Írja le, mit szeretne tenni az PicoClaw-val',
      newConversation: 'Új beszélgetés',
      processing: 'Feldolgozás...',
      agent: {
        defaultTitle: 'Általános asszisztens',
        defaultDescription: 'Általános csevegési, keresési és munkaterületi súgó.',
        kvmTitle: 'Távoli vezérlés',
        kvmDescription: 'Működtesse a távoli gazdagépet az NanoKVM segítségével.',
        switched: 'Ügynöki szerepkör megváltozott',
        switchFailed: 'Nem sikerült váltani az ügynöki szerepkört'
      },
      send: 'Küldés',
      cancel: 'Mégse',
      status: {
        connecting: 'Csatlakozás az átjáróhoz...',
        connected: 'PicoClaw munkamenet csatlakoztatva',
        disconnected: 'PicoClaw munkamenet lezárva',
        stopped: 'Leállítási kérés elküldve',
        runtimeStarted: 'PicoClaw Runtime elindult',
        runtimeStartFailed: 'Nem sikerült elindítani a PicoClaw Runtime-ot',
        runtimeStopped: 'PicoClaw Runtime leállt',
        runtimeStopFailed: 'Nem sikerült leállítani a PicoClaw Runtime-ot',
        controlSwitchedToMCP: 'A vezérlés átkerült a külső MCP-szolgáltatáshoz'
      },
      connection: {
        runtime: {
          checking: 'Ellenőrzés',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime kész',
          stopped: 'Runtime leállt',
          blockedByMCP: 'A külső MCP-vezérlés aktív',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime nem érhető el',
          configError: 'Konfigurációs hiba'
        },
        transport: {
          connecting: 'Csatlakozás',
          connected: 'Csatlakoztatva',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Üresjárat',
          busy: 'Elfoglalt'
        }
      },
      message: {
        toolAction: 'Akció',
        observation: 'Megfigyelés',
        screenshot: 'Képernyőkép'
      },
      overlay: {
        locked: 'PicoClaw vezérli az eszközt. A kézi bevitel szünetel.'
      },
      control: {
        picoclaw: 'Eszközvezérlés: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Eszközvezérlés: külső MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Eszközvezérlés: kikapcsolva',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Vezérlés átadása',
        release: 'Vezérlés feloldása',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw-vezérlés megadva',
        released: 'PicoClaw-vezérlés feloldva',
        grantFailed: 'Nem sikerült megadni a PicoClaw-vezérlést',
        releaseFailed: 'Nem sikerült feloldani a PicoClaw-vezérlést',
        grantConfirmTitle: 'Átváltja az eszközvezérlést PicoClaw-ra?',
        grantConfirmDesc: 'A külső MCP eszközírásai megszakadnak.'
      },
      install: {
        install: 'PicoClaw telepítése',
        installing: 'PicoClaw telepítése folyamatban',
        success: 'PicoClaw sikeresen telepítve',
        failed: 'Nem sikerült telepíteni PicoClaw',
        uninstalling: 'Runtime eltávolítása...',
        uninstalled: 'A Runtime sikeresen eltávolítva.',
        uninstallFailed: 'Az eltávolítás nem sikerült.',
        requiredTitle: 'PicoClaw nincs telepítve',
        requiredDescription:
          'Telepítse a PicoClaw alkalmazást a PicoClaw Runtime elindítása előtt.',
        progressDescription: 'PicoClaw letöltése és telepítése folyamatban van.',
        stages: {
          preparing: 'Felkészülés',
          downloading: 'Letöltés',
          extracting: 'Kibontás',
          verifying: 'Ellenőrzés',
          installing: 'Telepítés folyamatban',
          installed: 'Telepítve',
          install_timeout: 'Időtúllépés',
          install_failed: 'Sikertelen'
        }
      },
      model: {
        requiredTitle: 'Modellkonfiguráció szükséges',
        requiredDescription: 'A PicoClaw chat használata előtt konfigurálja az PicoClaw modellt.',
        docsTitle: 'Konfigurációs útmutató',
        docsDesc: 'Támogatott modellek és protokollok',
        menuLabel: 'Modell konfigurálása',
        modelIdentifier: 'Modellazonosító',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API-kulcs',
        apiKeyPlaceholder: 'Adja meg a modell API-kulcsát',
        save: 'Mentés',
        saving: 'Mentés',
        saved: 'A modell konfigurációja mentve',
        saveFailed: 'Nem sikerült menteni a modellkonfigurációt',
        invalid: 'A modellazonosító, az API Base URL és az API-kulcs megadása kötelező'
      },
      uninstall: {
        menuLabel: 'Eltávolítás',
        confirmTitle: 'Eltávolítás PicoClaw',
        confirmContent:
          'Biztosan eltávolítja a következőt: PicoClaw? Ezzel törli a végrehajtható fájlt és az összes konfigurációs fájlt.',
        confirmOk: 'Eltávolítás',
        confirmCancel: 'Mégse'
      },
      history: {
        title: 'Előzmények',
        loading: 'Munkamenetek betöltése...',
        emptyTitle: 'Még nincs előzmény',
        emptyDescription: 'A korábbi PicoClaw munkamenetek itt jelennek meg.',
        loadFailed: 'Nem sikerült betölteni a munkamenet-előzményeket',
        deleteFailed: 'Nem sikerült törölni a munkamenetet',
        deleteConfirmTitle: 'Munkamenet törlése',
        deleteConfirmContent: 'Biztos, hogy törölni szeretné a következőt: "{{title}}"?',
        deleteConfirmOk: 'Törlés',
        deleteConfirmCancel: 'Mégse',
        messageCount_one: '{{count}} üzenet',
        messageCount_other: '{{count}} üzenet',
        messageCount: '{{count}} üzenet'
      },
      config: {
        startRuntime: 'PicoClaw indítása',
        stopRuntime: 'PicoClaw leállítása'
      },
      start: {
        enableConfirmTitle: 'Átváltja a vezérlést a PicoClaw-ra?',
        enableConfirmDesc: 'A PicoClaw indítása letiltja a külső MCP-szolgáltatást.',
        enableConfirmOk: 'PicoClaw indítása',
        enableConfirmCancel: 'Mégse',
        title: 'PicoClaw indítása',
        description: 'Indítsa el a Runtime-ot a PicoClaw segéd használatának megkezdéséhez.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Problémába ütköztünk',
      refresh: 'Frissítés'
    },
    fullscreen: {
      toggle: 'Teljes képernyő váltás'
    },
    menu: {
      collapse: 'Menü összecsukása',
      expand: 'Bontsa ki a menüt'
    }
  }
};

export default hu;
