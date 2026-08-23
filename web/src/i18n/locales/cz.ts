const cz = {
  translation: {
    head: {
      desktop: 'Vzdálená plocha',
      login: 'Přihlášení',
      changePassword: 'Změna hesla',
      terminal: 'Terminál',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Přihlášení',
      placeholderUsername: 'Zadejte prosím uživatelské jméno',
      placeholderPassword: 'Zadejte prosím heslo',
      placeholderPassword2: 'Zadejte prosím heslo znovu',
      noEmptyUsername: 'Uživatelské jméno nesmí být prázdné',
      noEmptyPassword: 'Heslo nesmí být prázdné',
      noAccount:
        'Nepodařilo se získat informace o uživateli, prosím obnovte stránku nebo resetujte heslo',
      invalidUser: 'Neplatné uživatelské jméno nebo heslo',
      locked: 'Příliš mnoho přihlášení, zkuste to znovu později',
      globalLocked: 'Systém je chráněn, zkuste to znovu později',
      error: 'Neočekávaná chyba',
      changePassword: 'Změnit heslo',
      changePasswordDesc:
        'Pro bezpečnost vašeho zařízení prosím změňte heslo pro přihlášení na webu.',
      differentPassword: 'Hesla se neshodují',
      illegalUsername: 'Uživatelské jméno obsahuje nepovolené znaky',
      illegalPassword: 'Heslo obsahuje nepovolené znaky',
      forgetPassword: 'Zapomenuté heslo',
      ok: 'OK',
      cancel: 'Zrušit',
      loginButtonText: 'Přihlášení',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'Podrobné kroky najdete v tomto dokumentu:',
        reset3: 'Výchozí webový účet:',
        reset4: 'Výchozí účet SSH:',
        change1: 'Upozorňujeme, že tato akce změní následující hesla:',
        change2: 'Heslo pro webové přihlášení',
        change3: 'Heslo systémového uživatele root (heslo pro přihlášení SSH)',
        change4: 'Chcete-li hesla resetovat, podržte tlačítko BOOT na NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Nastavit Wi-Fi pro NanoKVM',
      success: 'Please check the network status of NanoKVM and visit the new IP address.',
      failed: 'Operace selhala, zkuste to znovu.',
      invalidMode:
        'Aktuální režim nepodporuje nastavení sítě. Přejděte do svého zařízení a povolte konfigurační režim Wi-Fi.',
      confirmBtn: 'Ok',
      finishBtn: 'Dokončeno',
      ap: {
        authTitle: 'Vyžaduje se ověření',
        authDescription: 'Pokračujte zadáním hesla AP',
        authFailed: 'Neplatné heslo AP',
        passPlaceholder: 'AP heslo',
        verifyBtn: 'Ověřte'
      }
    },
    screen: {
      scale: 'Měřítko',
      title: 'Obrazovka',
      video: 'Režim videa',
      videoDirectTips: 'Chcete-li používat tento režim, povolte HTTPS v "Nastavení > Zařízení"',
      resolution: 'Rozlišení',
      auto: 'Automatické',
      autoTips:
        'Může docházet k trhání obrazu nebo posunu myši při určitých rozlišeních. Zvažte úpravu rozlišení vzdáleného hostitele nebo vypněte automatický režim.',
      fps: 'FPS',
      customizeFps: 'Přizpůsobit',
      quality: 'Kvalita',
      qualityLossless: 'Bezeztrátový',
      qualityHigh: 'Vysoký',
      qualityMedium: 'Střední',
      qualityLow: 'Nízký',
      frameDetect: 'Detekce snímků',
      frameDetectTip:
        'Vypočítá rozdíl mezi snímky. Přenos video streamu se zastaví, pokud nejsou detekovány změny na obrazovce vzdáleného hostitele.',
      resetHdmi: 'Resetovat HDMI',
      mixedH264: {
        title: 'Konflikt streamu H.264',
        description:
          'H.264 Direct a H.264 WebRTC se používají současně. To může způsobit trhání obrazu nebo poškozené video. Používejte pouze jeden režim H.264.'
      },
      webrtcConnectionFailed: {
        title: 'Připojení WebRTC se nezdařilo',
        description: 'Zkontrolujte síťové připojení nebo přepněte režim videa.'
      },
      captureStatus: {
        hdmiError: 'Chyba obrazu HDMI',
        unsupportedResolution: 'Aktuální rozlišení není podporováno',
        retrieving: 'Načítá se obraz...',
        changingResolution: 'Přepíná se rozlišení...',
        updateFailed: 'Obraz se teď nemůže aktualizovat',
        videoError: 'Chyba zobrazení videa',
        noHdmi: 'Nebyl zjištěn signál HDMI',
        unavailable: 'Obraz teď nelze zobrazit'
      }
    },
    keyboard: {
      title: 'Klávesnice',
      paste: 'Vložit',
      tips: 'Podporovány jsou pouze standardní písmena a symboly klávesnice',
      placeholder: 'Zadejte text',
      submit: 'Odeslat',
      virtual: 'Klávesnice',
      readClipboard: 'Přečíst ze schránky',
      clipboardPermissionDenied:
        'Oprávnění ke schránce odepřeno. Povolte prosím přístup do schránky ve svém prohlížeči.',
      clipboardReadError: 'Nepodařilo se přečíst schránku',
      dropdownEnglish: 'anglicky',
      dropdownGerman: 'německy',
      dropdownFrench: 'francouzsky',
      dropdownRussian: 'rusky',
      shortcut: {
        title: 'Zkratky',
        custom: 'Vlastní',
        capture: 'Kliknutím sem zachytíte zástupce',
        clear: 'Jasno',
        save: 'Uložit',
        captureTips:
          'Zachycení systémových kláves (například klávesy Windows) vyžaduje oprávnění pro celou obrazovku.',
        enterFullScreen: 'Přepnout režim celé obrazovky.'
      },
      leaderKey: {
        title: 'Klávesa Leader',
        desc: 'Obejít omezení prohlížeče a odeslat systémové zkratky přímo vzdálenému hostiteli.',
        howToUse: 'Jak používat',
        simultaneous: {
          title: 'Simultánní režim',
          desc1: 'Stiskněte a podržte klávesu Leader a poté stiskněte zkratku.',
          desc2: 'Intuitivní, ale může být v rozporu se systémovými zkratkami.'
        },
        sequential: {
          title: 'Sekvenční režim',
          desc1:
            'Stiskněte klávesu Leader → postupně stiskněte zkratku → znovu stiskněte klávesu Leader.',
          desc2: 'Vyžaduje více kroků, ale zcela se vyhne systémovým konfliktům.'
        },
        enable: 'Povolit klávesu Leader',
        tip: 'Když je tato klávesa nastavena jako klávesa Leader, slouží pouze jako spouštěč zkratek a ztrácí své výchozí chování.',
        placeholder: 'Stiskněte klávesu Leader',
        shiftRight: 'Pravý Shift',
        ctrlRight: 'Pravý Ctrl',
        metaRight: 'Pravý Win',
        submit: 'Odeslat',
        recorder: {
          rec: 'REC',
          activate: 'Aktivovat klávesy',
          input: 'Stiskněte prosím zkratku...'
        }
      }
    },
    mouse: {
      title: 'Myš',
      cursor: 'Styl kurzoru',
      default: 'Výchozí kurzor',
      pointer: 'Ukazovací kurzor',
      cell: 'Kurzor buňky',
      text: 'Textový kurzor',
      grab: 'Chytnout kurzor',
      hide: 'Skrýt kurzor',
      mode: 'Režim myši',
      absolute: 'Absolutní režim',
      relative: 'Relativní režim',
      direction: 'Směr kolečka',
      scrollUp: 'Přejděte nahoru',
      scrollDown: 'Přejděte dolů',
      speed: 'Rychlost kolečka',
      fast: 'Rychle',
      slow: 'Pomalu',
      requestPointer:
        'Používá se relativní režim. Klikněte prosím na plochu pro získání kurzoru myši.',
      resetHid: 'Resetovat HID',
      hidOnly: {
        title: 'Režim pouze HID',
        desc: 'Pokud vaše myš a klávesnice přestanou reagovat a resetování HID nepomůže, může jít o problém s kompatibilitou mezi NanoKVM a zařízením. Zkuste povolit režim HID-Only pro lepší kompatibilitu.',
        tip1: 'Povolení režimu HID-Only odpojí virtuální U-disk a virtuální síť',
        tip2: 'V režimu HID-Only je připojení obrazu zakázáno',
        tip3: 'NanoKVM se po přepnutí režimů automaticky restartuje',
        enable: 'Povolit režim HID-Only',
        disable: 'Zakázat režim HID-Only'
      }
    },
    image: {
      title: 'Obrázky',
      loading: 'Načítání...',
      empty: 'Nic nenalezeno',
      mountMode: 'Režim připojení',
      mountFailed: 'Připojení se nezdařilo',
      mountDesc:
        'V některých systémech je nutné před připojením obrazu vysunout virtuální disk na vzdáleném hostiteli.',
      unmountFailed: 'Odpojení se nezdařilo',
      unmountDesc:
        'Na některých systémech se musíte před odpojením obrazu ručně vysunout ze vzdáleného hostitele.',
      refresh: 'Obnovte seznam obrázků',
      attention: 'Pozor',
      deleteConfirm: 'Opravdu chcete smazat tento obrázek?',
      okBtn: 'Ano',
      cancelBtn: 'Ne',
      tips: {
        title: 'Jak nahrát',
        usb1: 'Připojte NanoKVM k vašemu počítači přes USB.',
        usb2: 'Ujistěte se, že je virtuální disk připojen (Nastavení - Virtuální disk).',
        usb3: 'Otevřete virtuální disk na vašem počítači a zkopírujte soubor s obrazem do kořenového adresáře virtuálního disku.',
        scp1: 'Ujistěte se, že jsou NanoKVM a váš počítač ve stejné místní síti.',
        scp2: 'Otevřete terminál na vašem počítači a použijte příkaz SCP pro nahrání souboru s obrazem do adresáře /data na zařízení NanoKVM.',
        scp3: 'Příklad: scp cesta-k-vašemu-obrazu root@ip-nanokvm:/data',
        tfCard: 'SD Karta',
        tf1: 'Tato metoda je podporována na systémech Linux',
        tf2: 'Vyjměte SD kartu z NanoKVM (u plné verze nejprve rozložte krabičku).',
        tf3: 'Vložte SD kartu do čtečky karet a připojte ji k vašemu počítači.',
        tf4: 'Zkopírujte soubor s obrazem do adresáře /data na SD kartě.',
        tf5: 'Vložte SD kartu zpět do NanoKVM.'
      }
    },
    script: {
      title: 'Skript',
      upload: 'Nahrát',
      run: 'Spustit',
      runBackground: 'Spustit na pozadí',
      runFailed: 'Spuštění se nezdařilo',
      attention: 'Pozor',
      delDesc: 'Opravdu chcete tento soubor smazat?',
      confirm: 'Ano',
      cancel: 'Ne',
      delete: 'Smazat',
      close: 'Zavřít'
    },
    terminal: {
      title: 'Terminál',
      nanokvm: 'Terminál NanoKVM',
      serial: 'Terminál sériového portu',
      serialPort: 'Sériový port',
      serialPortPlaceholder: 'Zadejte prosím sériový port',
      baudrate: 'Přenosová rychlost',
      parity: 'Parita',
      parityNone: 'Žádné',
      parityEven: 'Sudá',
      parityOdd: 'Lichá',
      flowControl: 'Řízení toku',
      flowControlNone: 'Žádné',
      flowControlSoft: 'Softwarové',
      flowControlHard: 'Hardwarové',
      dataBits: 'Datové bity',
      stopBits: 'Stop bity',
      confirm: 'OK'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Odesílání příkazu...',
      sent: 'Příkaz odeslán',
      input: 'Zadejte prosím MAC adresu',
      ok: 'OK'
    },
    download: {
      title: 'Stahovač obrazů',
      input: 'Zadejte prosím vzdálený obrázek URL',
      ok: 'OK',
      disabled: 'Oddíl /data je RO, takže obrázek nelze stáhnout',
      uploadbox: 'Přetáhněte soubor sem nebo kliknutím vyberte',
      inputfile: 'Zadejte soubor obrázku',
      NoISO: 'Žádné ISO',
      sha256: 'SHA-256 (volitelné)',
      sha256Placeholder: 'Zadejte 64znakový kontrolní součet SHA-256',
      invalidSHA256: 'SHA-256 musí být 64znakový hexadecimální řetězec',
      failed: 'Stažení se nezdařilo',
      success: 'Stažení proběhlo úspěšně',
      checksumFailed: 'Stažení se nezdařilo: ověření SHA-256 selhalo',
      cancel: 'Zrušit',
      cancelFailed: 'Stažení se nepodařilo zrušit'
    },
    power: {
      title: 'Napájení',
      showConfirm: 'Potvrzení',
      showConfirmTip: 'Výkonové operace vyžadují další potvrzení',
      reset: 'Resetovat',
      power: 'Napájení',
      powerShort: 'Napájení (krátký stisk)',
      powerLong: 'Napájení (dlouhý stisk)',
      resetConfirm: 'Pokračovat v operaci resetování?',
      powerConfirm: 'Pokračovat v napájení?',
      okBtn: 'Ano',
      cancelBtn: 'Ne'
    },
    devices: {
      title: 'Zařízení',
      stale: 'Živý stav zařízení není dostupný. Probíhá opětovné připojení.',
      empty: 'Není nakonfigurován žádný slot pro kameru ani mikrofon.',
      available: 'Dostupné',
      waiting: 'Hostitel čeká na zdroj',
      hostOpen: 'Hostitel otevřen',
      hostIdle: 'Hostitel nečinný',
      sending: 'Odesílá se z tohoto prohlížeče',
      black: 'Černé video',
      silence: 'Digitální ticho',
      resuming: 'Čeká na obnovení',
      stop: 'Ukončit sdílení',
      disconnect: 'Odpojit',
      takeover: 'Převzít',
      refused: 'Používá {{owner}} ze zdroje {{source}}',
      connectedSources_one: '{{count}} připojený zdroj',
      connectedSources_other: '{{count}} připojených zdrojů',
      connectedSources: '{{count}} připojených zdrojů',
      connection: {
        connecting: 'Připojování',
        connected: 'Živě',
        disconnected: 'Opětovné připojování'
      },
      share: {
        camera: 'Sdílet kameru',
        microphone: 'Sdílet mikrofon',
        usbDevice: 'Sdílet USB'
      },
      permission: {
        denied: 'Blokováno v nastavení webu ve vašem prohlížeči',
        prompt: 'Prohlížeč se zeptá na přístup'
      },
      mic: {
        mute: 'Ztlumit',
        unmute: 'Zrušit ztlumení'
      },
      revoked: {
        released: 'Sdílení bylo ukončeno',
        lease_expired: 'Zápůjčka vypršela dříve, než se tento prohlížeč vrátil',
        admin_disconnect: 'Správce odpojil všechny zdroje',
        slot_removed: 'Slot byl odebrán',
        slot_changed: 'Slot byl překonfigurován',
        taken_over: 'Správce tento slot převzal'
      },
      usb: {
        surrendered: 'USB passthrough drží klávesnici a myš',
        surrenderedDesc:
          'Vzdálený host vidí importované zařízení místo klávesnice, myši a virtuálních médií NanoKVM. Vrátí se, jakmile relace skončí.',
        unsupported: 'WebUSB vyžaduje prohlížeč založený na Chromiu přes HTTPS',
        session: 'Předává se {{device}} ({{mode}})',
        idle: 'Žádná relace passthrough',
        mode: {
          hybrid: 'hybridní',
          exact: 'přesný'
        }
      }
    },
    settings: {
      title: 'Nastavení',
      display: {
        title: 'Obrazovka',
        loading: 'Načítání...',
        active: 'Aktivní EDID',
        activeUnknown:
          'NanoKVM od spuštění nezapsal žádné EDID, takže není známo, jaký monitor hostitel vidí.',
        appliedAt: 'Použito {{time}}',
        download: 'Stáhnout',
        downloadBackup: 'Stáhnout předchozí',
        preset: 'Předvolba monitoru',
        presetPlaceholder: 'Vyberte monitor',
        upload: 'Nahrát',
        selected: 'Vybrané EDID',
        errors: 'Chyby',
        warnings: 'Upozornění',
        info: 'Informace',
        unknownMonitor: 'Neznámý monitor',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Zvuk',
        audioNo: 'Bez zvuku',
        extensionBlocks: 'Rozšiřující bloky: {{blocks}}',
        apply: 'Použít',
        applyTitle: 'Použít toto EDID?',
        before: 'Současné',
        after: 'Nové',
        hdmiNotice: 'Během zápisu EDID se snímání obrazu zastaví a poté se samo znovu spustí.',
        powerCycleNotice:
          'Aby se nové EDID projevilo, je nutné zařízení fyzicky odpojit od napájení a znovu připojit.',
        powerCycleUnverified:
          'Zápis se nepodařilo ověřit, takže videočip si ponechá to, co v sobě právě má, dokud toto zařízení fyzicky neodpojíte od napájení a znovu nepřipojíte.',
        applied: 'EDID použito a ověřeno.',
        applyFailed: 'Použití EDID selhalo.',
        busy: 'Video čip byl zaneprázdněn. Zkuste to znovu.',
        unsupported: 'Toto zařízení neumožňuje změnu EDID.',
        toolMissing: 'V tomto firmwaru chybí nástroj pro EDID.',
        noAudio: 'Toto EDID neohlašuje zvuk, hostitel proto může přestat posílat zvuk.',
        oldVersion: 'Toto EDID používá verzi starší než 1.4.',
        interlaced: 'Preferované časování je prokládané.',
        tooLarge:
          'Preferované časování je vyšší než 1920x1080 při 60 Hz, tedy více, než dokáže NanoKVM zachytit.',
        recovery: 'Obnovení',
        recoveryNeeded:
          'Poslední zápis se nepodařilo ověřit, takže oblast EDID ve video čipu je v neznámém stavu. Obnovte tovární EDID, aby byl stav opět známý.',
        recoveryDesc:
          'Zapište do video čipu známé EDID, pokud po použití jiného zůstal hostitel bez obrazu.',
        restoreFactory: 'Obnovit tovární EDID',
        restoreBackup: 'Obnovit předchozí EDID',
        restoreTitle: 'Obnovit toto EDID?',
        restoreFactoryTarget: 'Tovární EDID dodávané s NanoKVM.',
        restoreBackupTarget: 'Nejnovější záloha, použitá {{time}}.',
        restoreNotice: 'Obnovení se zapisuje stejně jako použití a má stejné důsledky.',
        restored: 'EDID bylo obnoveno a ověřeno.',
        restoreFailed: 'Obnovení EDID se nezdařilo.',
        mismatchTitle: 'Zapsáno a přečteno zpět',
        mismatchWritten: 'Zapsáno',
        mismatchRead: 'Přečteno zpět',
        restoreOkBtn: 'Obnovit',
        hardware: 'Zjištěný hardware: {{hardware}}',
        hardwareUnknown: 'Neznámý',
        confirmWord: 'POTVRDIT',
        confirmPrompt: 'Pro odemknutí tlačítka pro použití napište {{word}}.',
        okBtn: 'Použít',
        cancelBtn: 'Zrušit'
      },
      presentation: {
        title: 'Prezentace USB',
        loading: 'Načítání...',
        current: 'Aktuální prezentace USB',
        noProfile: 'Není použit žádný profil',
        linked: 'Propojené funkce',
        hostState: 'USB hostitele',
        hostUnbound: 'Řadič není navázán',
        hdmiState: 'Vstup HDMI',
        hdmiSignal: 'Signál je přítomen',
        hdmiUnreported: 'Zatím žádné hlášení o zachytávání',
        endpoints: 'Endpointy',
        fifos: 'Sloty FIFO',
        pending: 'Nevyřízené změny',
        pendingEdits: 'Neuložené úpravy identity',
        pendingProfile: '{{profile}} je vybrán, ale není použit',
        pendingNone: 'Žádné',
        lastApply: 'Poslední použití',
        applyFailed: 'Selhalo u {{profile}} v {{time}}',
        applyClean: 'Není zaznamenáno žádné selhání',
        lastKnownGood: 'Poslední známý funkční',
        rollbackTarget: 'Cíl návratu',
        rollbackNone: 'Žádný',
        powerCyclePending:
          'Řadič byl hostiteli odebrán. Chcete-li zařízení získat zpět, vypněte a znovu zapněte připojený počítač.',
        rollback: 'Vrátit zpět',
        rollbackTitle: 'Vrátit se k profilu {{profile}}?',
        rollbackDesc: 'Gadget se znovu vyčíslí; USB funkce na okamžik vypadnou.',
        profile: 'Profil USB',
        builtIn: 'vestavěný',
        descriptors: 'deskriptory',
        clone: 'Klonovat',
        cloneTitle: 'Klonovat tento profil',
        cloneToEdit:
          'Vestavěné profily zůstávají jen pro čtení. Naklonujte tento profil, chcete-li upravit jeho identitu.',
        profileName: 'Název profilu',
        profileNameHint: 'Malá písmena, číslice, tečky, podtržítka a pomlčky.',
        import: 'Importovat balíček',
        export: 'Exportovat balíček',
        delete: 'Smazat',
        deleteTitle: 'Smazat tento profil?',
        deleteDesc: 'Smazat {{profile}} z NanoKVM.',
        identity: 'Identita USB',
        preset: 'Předvolená identita',
        presetPlaceholder: 'Převzít identitu ze známého zařízení',
        presetHint:
          'Předvolba vyplní Vendor ID, Product ID a obě pole s názvem. Žádné deskriptory s sebou nenese.',
        presetSource: 'Identita tak, jak je zaznamenána v {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Toto Vendor ID patří jinému výrobci',
        productId: 'Product ID',
        bcdUSB: 'Verze USB',
        bcdDevice: 'Verze zařízení',
        manufacturer: 'Výrobce',
        product: 'Produkt',
        serial: 'Sériové číslo',
        configuration: 'Konfigurační řetězec',
        functions: 'Funkce',
        descriptorAssets: 'Uložené soubory deskriptorů: {{count}}',
        endpointUse:
          'IN {{inUse}} použito, {{inFree}} volných; OUT {{outUse}} použito, {{outFree}} volných',
        preview: 'Ověřit',
        save: 'Uložit',
        apply: 'Použít',
        applyTitle: 'Použít tento profil USB?',
        applyDesc: 'NanoKVM bude připojenému počítači předkládat {{profile}}.',
        reconnect:
          'Klávesnice, myš a další funkce USB se na okamžik odpojí, než se gadget znovu naváže.',
        applyLinks: 'Propojí: {{functions}}',
        applyRemoves: 'Odebere: {{functions}}',
        applyNoHid:
          'Po tomto použití nezůstane žádná funkce HID. Klávesnice a myš přestanou fungovat.',
        applyRollback: 'Neúspěšné použití se vrátí k profilu {{profile}}.',
        recoveryPowerCycle:
          'Toto použití nepřežije žádné HID, takže hostitele, který přestane reagovat, lze obnovit jen vypnutím a zapnutím napájení.',
        recoveryReboot:
          'Ze složeného zařízení zmizí jedno rozhraní; hostitel může potřebovat restart, aby zbytek znovu navázal.',
        recoveryHdmiReset:
          'Videofunkce se vytvoří znovu, takže se resetuje i řetězec zachytávání za ní.',
        recoveryReconnect: 'Hostitel zařízení znovu vyčíslí; USB funkce na okamžik vypadnou.',
        cancel: 'Zrušit',
        noFunctions: 'Žádné propojené funkce',
        loadFailed: 'Profily prezentace se nepodařilo načíst',
        operationFailed: 'Operace s prezentací selhala'
      },
      passthrough: {
        title: 'Průchod USB',
        loading: 'Načítání...',
        hidWarning: 'Spuštění průchodu odevzdá klávesnici, myš i virtuální média',
        hidWarningDesc:
          'NanoKVM má jediný řadič USB zařízení a proxy jej potřebuje celý. Během relace proto vzdálený hostitel vidí předané zařízení místo klávesnice, myši a virtuálních médií NanoKVM. Vrátí se samy ve chvíli, kdy relaci zastavíte. Tohoto webového rozhraní se to netýká, takže relaci můžete z této stránky kdykoli ukončit.',
        isoWarning: 'Webkamery, mikrofony a další izochronní zařízení předat nelze',
        isoWarningDesc:
          'Tento hardware přenáší pouze řídicí, hromadné a přerušovací přenosy. Zvuková a obrazová zařízení fungovat nebudou, ať je připojíte jakkoli.',
        session: 'Relace',
        activeDesc: 'Zařízení je importováno a proxy drží řadič USB.',
        inactiveDesc: 'Neběží žádná relace. Klávesnice, myš i virtuální média fungují normálně.',
        device: 'Zařízení',
        busId: 'ID sběrnice',
        speed: 'Rychlost',
        exporter: 'Exportér',
        local: 'Importováno jako',
        localValue: 'Sběrnice {{bus}}, adresa {{address}}',
        udc: 'Řadič USB',
        pid: 'PID proxy',
        startedAt: 'Spuštěno',
        isoDevice:
          'Toto zařízení hlásí třídu zvuku nebo videa, která vyžaduje izochronní přenosy. Fungovat nebude.',
        exporterLabel: 'Adresa exportéra',
        exporterHint:
          'Hostitel a port, na které se NanoKVM připojuje. Přes tunel níže je to {{exporter}}.',
        busIdLabel: 'ID sběrnice na vašem počítači',
        busIdHint: 'Busid, které usbip list -l pro zařízení vypíše, například {{example}}.',
        start: 'Spustit průchod',
        stop: 'Zastavit průchod',
        startTitle: 'Spustit průchod USB?',
        startDevice: 'NanoKVM importuje {{busId}} z {{exporter}}.',
        startHid:
          'Klávesnice USB, myš i virtuální média přestanou po dobu relace fungovat a samy se rozběhnou, jakmile ji zastavíte.',
        startIso:
          'Webkamery, mikrofony a další izochronní zařízení na tomto hardwaru fungovat nebudou.',
        startWeb:
          'Toto webové rozhraní funguje dál, relaci tedy můžete z této stránky kdykoli zastavit.',
        okBtn: 'Spustit',
        cancelBtn: 'Zrušit',
        instructions: 'Na vašem počítači',
        instructionsDesc:
          'Záměrně se neinstaluje žádný klientský agent. Spusťte tyto běžné příkazy usbip na počítači, ke kterému je zařízení připojeno.',
        copyFailed: 'Kopírování se nezdařilo. Zkopírujte příkaz ručně.',
        directNote:
          'Bez tunelu musí být usbipd dostupný ve vaší síti a adresa exportéra výše na něj musí ukazovat. usbip přenáší zařízení nešifrovaně, proto je tunel vhodnější.',
        steps: {
          modprobe: {
            title: 'Načtěte ovladač na straně exportéra',
            desc: 'usbip-host umožňuje jádru předat místní zařízení. Ve výchozím stavu se nenačítá.'
          },
          list: {
            title: 'Najděte zařízení',
            desc: 'Vypíše všechna místní zařízení s jejich busid a dvojicí výrobce:produkt. Poznamenejte si busid toho, které chcete.'
          },
          bind: {
            title: 'Připojte je k usbip',
            desc: 'Odebere zařízení jeho běžnému ovladači, takže na tomto počítači přestane fungovat, dokud je neodpojíte.'
          },
          serve: {
            title: 'Nabídněte je',
            desc: 'usbipd zůstane v popředí a čeká, až NanoKVM zařízení importuje.',
            notice:
              'Běžný usbipd nemá volbu naslouchací adresy a naslouchá na všech rozhraních. Nechte port {{port}} na firewallu zavřený a pusťte k němu jen tunel níže.'
          },
          tunnel: {
            title: 'Nasměrujte je na NanoKVM',
            desc: 'Zpětný tunel SSH: port {{port}} na smyčce samotného NanoKVM se stane usbipd na tomto počítači. Nechte jej běžet po celou relaci.'
          },
          exporter: {
            title: 'Použijte toto jako exportéra',
            desc: 'Vložte to výše do pole exportéra, zadejte ID sběrnice a spusťte relaci.'
          },
          unbind: {
            title: 'Vraťte zařízení',
            desc: 'Po zastavení relace tímto zařízení vrátíte jeho běžnému ovladači na tomto počítači.'
          }
        }
      },
      mcp: {
        title: 'Služba MCP',
        service: 'Vzdálené ovládání MCP',
        serviceDesc:
          'Umožnit důvěryhodným klientům MCP ovládat klávesnici a myš a pořizovat snímky obrazovky',
        securityWarning:
          'Kdokoli s tímto API klíčem může ovládat vzdálený hostitel a zobrazit jeho obrazovku. Používejte HTTPS a povolte službu pouze v důvěryhodných sítích.',
        endpoint: 'Koncový bod',
        apiKey: 'API klíč',
        regenerateConfirmTitle: 'Vygenerovat nový MCP API klíč?',
        regenerateConfirmDesc: 'Aktuální klíč přestane okamžitě fungovat.',
        enableConfirmTitle: 'Povolit externí ovládání MCP?',
        enableConfirmDesc:
          'Povolením MCP se zastaví PicoClaw a ukončí se všechny aktivní relace PicoClaw.',
        failed: 'Operace MCP se nezdařila',
        copyFailed: 'Kopírování se nezdařilo. Zkopírujte ručně.',
        okBtn: 'Potvrdit',
        cancelBtn: 'Zrušit'
      },
      about: {
        title: 'O NanoKVM',
        information: 'Informace',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Verze aplikace',
        applicationTip: 'Verze webové aplikace NanoKVM',
        image: 'Verze obrazu',
        imageTip: 'Verze systémového obrazu NanoKVM',
        deviceKey: 'Klíč zařízení',
        community: 'Komunita',
        hostname: 'Název hostitele',
        hostnameUpdated: 'Název hostitele byl aktualizován. Pro použití restartujte.',
        ipType: {
          Wired: 'Kabelové',
          Wireless: 'Bezdrátové',
          Other: 'Jiné'
        }
      },
      appearance: {
        title: 'Vzhled',
        display: 'Zobrazení',
        language: 'Jazyk',
        languageDesc: 'Vyberte jazyk rozhraní',
        webTitle: 'Název webu',
        webTitleDesc: 'Přizpůsobte název webové stránky',
        menuBar: {
          title: 'Panel nabídek',
          mode: 'Režim zobrazení',
          modeDesc: 'Zobrazení panelu nabídek na obrazovce',
          modeOff: 'Vypnuto',
          modeAuto: 'Automatické skrytí',
          modeAlways: 'Vždy viditelné',
          keyboardLedStatus: 'Indikátory zámku klávesnice',
          keyboardLedStatusDesc:
            'Zobrazit stav Num Lock, Caps Lock a Scroll Lock vzdáleného počítače',
          icons: 'Ikony podnabídky',
          iconsDesc: 'Zobrazení ikon podnabídky na liště nabídek'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Stav zámků vzdálené klávesnice',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Zapnuto',
        off: 'Vypnuto',
        unknown: 'Neznámé'
      },
      device: {
        title: 'Zařízení',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Nikdy',
          15: '15 sec',
          30: '30 sec',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 hodina'
        },
        ssh: {
          description: 'Povolit vzdálený přístup SSH',
          tip: 'Před povolením nastavte silné heslo (Účet – Změnit heslo)'
        },
        advanced: 'Pokročilá nastavení',
        swap: {
          title: 'Vyměnit',
          disable: 'Zakázat',
          description: 'Nastavte velikost odkládacího souboru',
          tip: 'Povolení této funkce může zkrátit životnost vaší SD karty!'
        },
        mouseJiggler: {
          title: 'Mouse Jiggler',
          description: 'Zabraňte spánku vzdáleného hostitele',
          disable: 'Zakázat',
          absolute: 'Absolutní režim',
          relative: 'Relativní režim'
        },
        mdns: {
          description: 'Povolit službu zjišťování mDNS',
          tip: 'Vypnutí, pokud to není potřeba'
        },
        hdmi: {
          description: 'Povolit výstup HDMI/monitor',
          idleTimeoutTitle: 'Časový limit nečinnosti snímání',
          idleTimeoutDescription: 'Zastavit snímání HDMI po době bez aktivních diváků',
          minutes: 'min'
        },
        autostart: {
          title: 'Nastavení automatického spuštění skriptů',
          description: 'Správa skriptů, které se spouštějí automaticky při spuštění systému',
          new: 'Nové',
          deleteConfirm: 'Opravdu chcete tento soubor smazat?',
          yes: 'Ano',
          no: 'Ne',
          scriptName: 'Název skriptu automatického spuštění',
          scriptContent: 'Obsah skriptu automatického spuštění',
          settings: 'Nastavení'
        },
        hidOnly: 'HID-Pouze režim',
        hidOnlyDesc: 'Zastavit emulaci virtuálních zařízení a zachovat pouze základní ovládání HID',
        disk: 'Virtuální disk',
        diskDesc: 'Mount virtual U-disk on the remote host',
        network: 'Virtuální síť',
        networkDesc: 'Připojit virtuální síťovou kartu na vzdáleném hostiteli',
        networkProtocol: 'Síťový protokol',
        networkProtocolDesc: 'NCM pro moderní systémy, RNDIS pro starší Windows',
        reboot: 'Restartujte',
        rebootDesc: 'Opravdu chcete restartovat NanoKVM?',
        okBtn: 'Ano',
        cancelBtn: 'Ne'
      },
      network: {
        title: 'Síť',
        wifi: {
          title: 'Wi-Fi',
          description: 'Nastavit Wi-Fi',
          apMode: 'Režim AP je povolen, připojte se k Wi-Fi naskenováním QR kódu',
          connect: 'Připojit Wi-Fi',
          connectDesc1: 'Zadejte SSID sítě a heslo',
          connectDesc2: 'Zadejte heslo pro připojení k této síti',
          disconnect: 'Opravdu chcete síť odpojit?',
          failed: 'Připojení se nezdařilo, zkuste to znovu.',
          ssid: 'Název',
          password: 'Heslo',
          joinBtn: 'Připojit',
          confirmBtn: 'OK',
          cancelBtn: 'Zrušit'
        },
        tls: {
          description: 'Povolit protokol HTTPS',
          tip: 'Upozornění: Použití HTTPS může zvýšit latenci, zejména v režimu videa MJPEG.'
        },
        bridge: {
          title: 'Síťový most',
          twoDevices:
            'Router uvidí NanoKVM a ovládaný počítač jako dvě samostatná zařízení, každé s vlastní adresou.',
          loading: 'Načítání...',
          state: 'Stav',
          states: {
            disabled: 'Vypnuto',
            enabled: 'Zapnuto',
            rolledBack: 'Vráceno zpět',
            failed: 'Selhalo',
            pending: 'Probíhá'
          },
          uplink: 'Uplink',
          ports: 'Porty',
          protocol: 'Protokol zařízení',
          up: 'aktivní',
          down: 'neaktivní',
          noLink: 'bez linky',
          enableTitle: 'Zapnout síťový most?',
          disableTitle: 'Vypnout síťový most?',
          reconnect:
            'Během přesunu adresy se správcovské spojení na chvíli přeruší a znovu naváže.',
          rollback: 'Pokud ověření selže, předchozí konfigurace se automaticky obnoví.',
          enableBtn: 'Zapnout',
          disableBtn: 'Vypnout',
          cancelBtn: 'Zrušit',
          interrupted: 'Spojení bylo během aplikování přerušeno. Probíhá opětovná kontrola stavu.',
          pendingNotice: 'Změna mostu stále probíhá nebo byla přerušena před dokončením.',
          revert: 'Obnovit předchozí konfiguraci',
          rolledBackNotice:
            'Poslední změna byla vrácena zpět a předchozí konfigurace byla obnovena.',
          verifyFailed: 'Ověření selhalo: {{gates}}',
          gates: {
            address: 'adresa',
            gateway: 'brána',
            inbound: 'příchozí spojení'
          },
          inboundWeak:
            'Kontrola příchozího spojení prošla jen proto, že se NanoKVM připojil sám k sobě. To dokazuje, že webová služba naslouchá a je dostupná lokálně, nikoli že k ní dorazí požadavek ze sítě.',
          noCarrier:
            'Na portu {{port}} není linka. Most nemá cestu do sítě, dokud se nepřipojí kabel.',
          loop: 'Router se učí i na portu {{port}}, takže tento port je druhou cestou do stejné sítě. Spanning tree je vypnutý, takže smyčku zde nic nepřeruší: odpojte jednu ze dvou cest.',
          failedNotice:
            'Poslední změnu se nepodařilo vrátit zpět. NanoKVM může být dostupné jen přes Wi-Fi AP nebo sériovou konzoli.'
        },
        dns: {
          title: 'DNS',
          description: 'Nastavit DNS servery pro NanoKVM',
          mode: 'Režim',
          dhcp: 'DHCP',
          manual: 'Ručně',
          add: 'Přidat DNS',
          save: 'Uložit',
          invalid: 'Zadejte platnou IP adresu',
          noDhcp: 'Momentálně není k dispozici žádné DHCP DNS',
          saved: 'Nastavení DNS uloženo',
          saveFailed: 'Nastavení DNS se nepodařilo uložit',
          unsaved: 'Neuložené změny',
          maxServers: 'Je povoleno maximálně {{count}} DNS serverů',
          dnsServers: 'DNS servery',
          dhcpServersDescription: 'DNS servery jsou automaticky získávány z DHCP',
          manualServersDescription: 'DNS servery lze upravit ručně',
          networkDetails: 'Podrobnosti sítě',
          interface: 'Rozhraní',
          ipAddress: 'IP adresa',
          subnetMask: 'Maska podsítě',
          router: 'Router',
          none: 'Žádné'
        }
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Optimalizace paměti',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Vyměňte paměť',
          tip: 'Pokud problémy přetrvávají i po povolení optimalizace paměti, zkuste povolit odkládací paměť. Tím se ve výchozím nastavení nastaví velikost odkládacího souboru na 256MB, kterou lze upravit v „Nastavení > Zařízení“.'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Načítání...',
        notInstall: 'Tailscale nebyl nalezen! Prosím nainstalujte.',
        install: 'Nainstalovat',
        installing: 'Instalace probíhá',
        failed: 'Instalace se nezdařila',
        retry: 'Obnovte stránku a zkuste to znovu. Nebo zkuste instalaci manuálně',
        download: 'Stáhnout',
        package: 'instalační balíček',
        unzip: 'a rozbalit ho',
        upTailscale: 'Nahrajte Tailscale do adresáře NanoKVM /usr/bin/',
        upTailscaled: 'Nahrajte Tailscaled do adresáře NanoKVM /usr/sbin/',
        refresh: 'Obnovit stránku',
        notRunning: 'Tailscale neběží. Chcete-li pokračovat, spusťte jej.',
        run: 'Spustit',
        notLogin:
          'Zařízení nebylo dosud spárováno. Přihlaste se prosím a spárujte toto zařízení s vaším účtem.',
        urlPeriod: 'Tento odkaz je platný po dobu 10 minut',
        login: 'Přihlášení',
        loginSuccess: 'Přihlášení úspěšné',
        enable: 'Povolit Tailscale',
        deviceName: 'Název zařízení',
        deviceIP: 'IP zařízení',
        account: 'Účet',
        logout: 'Odhlásit se',
        logoutDesc: 'Opravdu se chcete odhlásit?',
        uninstall: 'Odinstalovat Tailscale',
        uninstallDesc: 'Opravdu chcete odinstalovat Tailscale?',
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
        loading: 'Načítání...',
        notInstall: 'Nenainstalováno',
        notConfigured: 'Nenakonfigurováno',
        stopped: 'Zastaveno',
        running: 'Běží',
        connected: 'Připojeno',
        error: 'Chyba',
        arguments: 'Argumenty',
        argumentsTip: 'Argumenty příkazové řádky předané službě při spuštění.',
        env: 'Proměnné prostředí',
        envKey: 'Název',
        envValue: 'Hodnota',
        envAdd: 'Přidat proměnnou',
        envRemove: 'Odebrat',
        configured: 'Nastaveno',
        save: 'Uložit',
        saved: 'Konfigurace uložena',
        start: 'Spustit',
        stop: 'Zastavit',
        restart: 'Restartovat',
        logs: 'Protokoly',
        logsEmpty: 'Zatím žádné záznamy',
        refresh: 'Obnovit',
        binary: 'Binární soubor',
        binaryShipped: 'Součást firmwaru',
        binaryCustom: 'Vlastní binární soubor',
        binaryUpload: 'Nahrát binární soubor',
        binaryRevert: 'Obnovit binární soubor z firmwaru',
        binaryRevertDesc: 'Smazat nahraný binární soubor a obnovit verzi dodanou s firmwarem?',
        serverWarning: 'Server bez omezení funguje jako otevřená proxy',
        noHealthSignal:
          'Tato služba nehlásí stav, takže NanoKVM ví jen to, že proces běží, nikoli zda je tunel připojen.',
        memoryWarning: 'Souběžný běh více služeb vzdáleného přístupu může vyčerpat paměť',
        okBtn: 'Ano',
        cancelBtn: 'Ne'
      },
      update: {
        title: 'Zkontrolovat aktualizaci',
        queryFailed: 'Nepodařilo se získat verzi',
        updateFailed: 'Aktualizace se nezdařila. Zkuste to prosím znovu.',
        isLatest: 'Máte nejnovější verzi.',
        available: 'Je dostupná aktualizace. Opravdu chcete aktualizovat?',
        updating: 'Aktualizace zahájena. Prosím čekejte...',
        confirm: 'Potvrdit',
        cancel: 'Zrušit',
        preview: 'Náhled aktualizací',
        previewDesc: 'Získejte včasný přístup k novým funkcím a vylepšením',
        previewTip:
          'Uvědomte si prosím, že předběžné verze mohou obsahovat chyby nebo neúplné funkce!',
        customServer: {
          title: 'Vlastní aktualizační server',
          desc: 'Vyhledávejte a stahujte online aktualizace ze zadaného serveru',
          invalidUrl:
            'Zadejte platnou adresu adresáře serveru HTTP nebo HTTPS bez parametrů, fragmentu nebo souboru latest.json.',
          loadFailed: 'Konfiguraci aktualizačního serveru se nepodařilo načíst.',
          saveFailed: 'Konfiguraci aktualizačního serveru se nepodařilo uložit.',
          saved: 'Konfigurace aktualizačního serveru byla uložena.',
          save: 'Uložit',
          confirmTitle: 'Použít vlastní aktualizační server?',
          confirmDesc:
            'SHA-512 pouze ověřuje, že balíček odpovídá manifestu poskytnutému tímto serverem. Neprokazuje, že je balíček oficiálním vydáním NanoKVM. Vadný nebo škodlivý server může způsobit nefunkčnost zařízení, ztrátu dat nebo narušení zabezpečení systému.',
          confirm: 'Přesto použít',
          previewDisabled:
            'Testovací aktualizace nejsou při použití vlastního aktualizačního serveru dostupné.'
        },
        offline: {
          title: 'Offline aktualizace',
          desc: 'Aktualizace prostřednictvím místního instalačního balíčku',
          upload: 'Nahrát',
          checksumPlaceholder: 'Kontrolní součet SHA-256 (volitelný)',
          invalidChecksum: 'Kontrolní součet SHA-256 musí obsahovat 64 hexadecimálních znaků.',
          checksumMismatch: 'Ověření SHA-256 se nezdařilo. Balíček může být poškozený.',
          invalidName: 'Neplatný formát souboru. Stáhněte si prosím z vydání GitHubu.',
          updateFailed: 'Aktualizace se nezdařila. Zkuste to prosím znovu.'
        }
      },
      account: {
        title: 'Účet',
        webAccount: 'Název webového účtu',
        password: 'Heslo',
        updateBtn: 'Update',
        logoutBtn: 'Odhlásit',
        logoutDesc: 'Opravdu se chcete odhlásit?',
        okBtn: 'Ano',
        cancelBtn: 'Ne'
      }
    },
    picoclaw: {
      title: 'PicoClaw asistent',
      empty: 'Otevřete panel a spusťte úlohu.',
      inputPlaceholder: 'Popište, co chcete, aby PicoClaw dělal',
      newConversation: 'Nová konverzace',
      processing: 'Zpracovává se...',
      agent: {
        defaultTitle: 'Obecný asistent',
        defaultDescription: 'Obecná nápověda pro chat, vyhledávání a pracovní prostor.',
        kvmTitle: 'Vzdálené ovládání',
        kvmDescription: 'Ovládejte vzdáleného hostitele prostřednictvím NanoKVM.',
        switched: 'Role agenta změněna',
        switchFailed: 'Přepnutí role agenta se nezdařilo'
      },
      send: 'Odeslat',
      cancel: 'Zrušit',
      status: {
        connecting: 'Připojování k bráně...',
        connected: 'Relace PicoClaw připojena',
        disconnected: 'Relace PicoClaw uzavřena',
        stopped: 'Požadavek na zastavení byl odeslán',
        runtimeStarted: 'Spuštěno běhové prostředí PicoClaw',
        runtimeStartFailed: 'Selhalo spuštění běhového prostředí PicoClaw',
        runtimeStopped: 'Běhové prostředí PicoClaw zastaveno',
        runtimeStopFailed: 'Zastavení běhového prostředí PicoClaw se nezdařilo',
        controlSwitchedToMCP: 'Ovládání bylo přepnuto na externí službu MCP'
      },
      connection: {
        runtime: {
          checking: 'Kontrola',
          restoring: 'Restoring PicoClaw',
          ready: 'Běhové prostředí připraveno',
          stopped: 'Běhové prostředí zastaveno',
          blockedByMCP: 'Externí ovládání MCP je aktivní',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Běhové prostředí není k dispozici',
          configError: 'Chyba konfigurace'
        },
        transport: {
          connecting: 'Připojování',
          connected: 'Připojeno',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Nečinný',
          busy: 'Zaneprázdněn'
        }
      },
      message: {
        toolAction: 'Akce',
        observation: 'Pozorování',
        screenshot: 'Snímek obrazovky'
      },
      overlay: {
        locked: 'PicoClaw ovládá zařízení. Ruční zadávání je pozastaveno.'
      },
      control: {
        picoclaw: 'Ovládání zařízení: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Ovládání zařízení: externí MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Ovládání zařízení: vypnuto',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Předat ovládání',
        release: 'Uvolnit',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'Ovládání PicoClaw povoleno',
        released: 'Ovládání PicoClaw uvolněno',
        grantFailed: 'Nepodařilo se předat ovládání PicoClaw',
        releaseFailed: 'Nepodařilo se uvolnit ovládání PicoClaw',
        grantConfirmTitle: 'Přepnout ovládání zařízení na PicoClaw?',
        grantConfirmDesc: 'Zápisy zařízení z externího MCP budou přerušeny.'
      },
      install: {
        install: 'Nainstalovat PicoClaw',
        installing: 'Instalace PicoClaw',
        success: 'PicoClaw úspěšně nainstalováno',
        failed: 'Nepodařilo se nainstalovat PicoClaw',
        uninstalling: 'Odinstalování běhového prostředí...',
        uninstalled: 'Běhové prostředí bylo úspěšně odinstalováno.',
        uninstallFailed: 'Odinstalace se nezdařila.',
        requiredTitle: 'PicoClaw není nainstalováno',
        requiredDescription: 'Nainstalujte PicoClaw před spuštěním běhového prostředí PicoClaw.',
        progressDescription: 'PicoClaw se stahuje a instaluje.',
        stages: {
          preparing: 'Příprava',
          downloading: 'Stahování',
          extracting: 'Extrakce',
          verifying: 'Ověřování',
          installing: 'Instalace probíhá',
          installed: 'Instalováno',
          install_timeout: 'Vypršel časový limit',
          install_failed: 'Selhalo'
        }
      },
      model: {
        requiredTitle: 'Je vyžadována konfigurace modelu',
        requiredDescription: 'Před použitím chatu PicoClaw nakonfigurujte model PicoClaw.',
        docsTitle: 'Průvodce konfigurací',
        docsDesc: 'Podporované modely a protokoly',
        menuLabel: 'Konfigurace modelu',
        modelIdentifier: 'Identifikátor modelu',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'Klíč API',
        apiKeyPlaceholder: 'Zadejte klíč API modelu',
        save: 'Uložit',
        saving: 'Ukládání',
        saved: 'Konfigurace modelu uložena',
        saveFailed: 'Nepodařilo se uložit konfiguraci modelu',
        invalid: 'Identifikátor modelu, API Base URL a klíč API jsou povinné'
      },
      uninstall: {
        menuLabel: 'Odinstalovat',
        confirmTitle: 'Odinstalovat PicoClaw',
        confirmContent:
          'Opravdu chcete odinstalovat PicoClaw? Tím smažete spustitelný soubor a všechny konfigurační soubory.',
        confirmOk: 'Odinstalovat',
        confirmCancel: 'Zrušit'
      },
      history: {
        title: 'Historie',
        loading: 'Načítání relací...',
        emptyTitle: 'Zatím žádná historie',
        emptyDescription: 'Zde se zobrazí předchozí relace PicoClaw.',
        loadFailed: 'Nepodařilo se načíst historii relace',
        deleteFailed: 'Smazání relace se nezdařilo',
        deleteConfirmTitle: 'Smazat relaci',
        deleteConfirmContent: 'Opravdu chcete smazat "{{title}}"?',
        deleteConfirmOk: 'Smazat',
        deleteConfirmCancel: 'Zrušit',
        messageCount_one: '{{count}} zpráva',
        messageCount_other: '{{count}} zpráv',
        messageCount: '{{count}} zpráv'
      },
      config: {
        startRuntime: 'Spustit PicoClaw',
        stopRuntime: 'Zastavit PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Přepnout ovládání na PicoClaw?',
        enableConfirmDesc: 'Spuštěním PicoClaw se deaktivuje externí služba MCP.',
        enableConfirmOk: 'Spustit PicoClaw',
        enableConfirmCancel: 'Zrušit',
        title: 'Spustit PicoClaw',
        description: 'Spusťte běhové prostředí a začněte používat asistenta PicoClaw.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Narazili jsme na problém',
      refresh: 'Obnovit'
    },
    fullscreen: {
      toggle: 'Přepnout na celou obrazovku'
    },
    menu: {
      collapse: 'Sbalit nabídku',
      expand: 'Rozbalte nabídku'
    }
  }
};

export default cz;
