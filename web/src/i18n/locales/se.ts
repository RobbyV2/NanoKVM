const se = {
  translation: {
    head: {
      desktop: 'Fjärrskrivbord',
      login: 'Logga in',
      changePassword: 'Byt lösenord',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Logga in',
      placeholderUsername: 'Användarnamn',
      placeholderPassword: 'Lösenord',
      placeholderCurrentPassword: 'Nuvarande lösenord',
      placeholderPassword2: 'Vänligen ange lösenordet igen',
      noEmptyUsername: 'Användarnamn krävs',
      noEmptyPassword: 'Lösenord krävs',
      passwordLength: 'Lösenordet måste vara mellan 8 och 72 tecken',
      noAccount: 'Kunde inte hämta användarinformation, uppdatera sidan eller återställ lösenordet',
      invalidUser: 'Ogiltigt användarnamn eller lösenord',
      locked: 'För många inloggningar, försök igen senare',
      globalLocked: 'System under skydd, försök igen senare',
      error: 'Oväntat fel',
      invalidCurrentPassword: 'Det nuvarande lösenordet är fel',
      changePassword: 'Byt lösenord',
      changePasswordDesc: 'För din enhets säkerhet, byt lösenord!',
      differentPassword: 'Lösenorden matchar inte',
      illegalUsername: 'Användarnamnet innehåller ogiltiga tecken',
      illegalPassword: 'Lösenordet innehåller ogiltiga tecken',
      forgetPassword: 'Glömt lösenord',
      ok: 'Ok',
      cancel: 'Avbryt',
      loginButtonText: 'Logga in',
      tips: {
        reset1: 'För att återställa lösenordet, håll in BOOT-knappen på NanoKVM i 10 sekunder.',
        reset2: 'För detaljerade steg, se detta dokument:',
        reset3: 'Standardkonto för webben:',
        reset4: 'Standardkonto för SSH:',
        change1: 'Observera att denna åtgärd ändrar följande lösenord:',
        change2: 'Webbinloggningslösenord',
        change3: 'Systemets root-lösenord (SSH-lösenord)',
        change4: 'För att återställa lösenordet, håll in BOOT-knappen på NanoKVM.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Konfigurera Wi-Fi för NanoKVM',
      success: 'Kontrollera nätverksstatusen för NanoKVM och besök den nya IP-adressen.',
      failed: 'Åtgärden misslyckades, försök igen.',
      invalidMode:
        'Det aktuella läget stöder inte nätverksinstallation. Gå till din enhet och aktivera Wi-Fi konfigurationsläge.',
      confirmBtn: 'Ok',
      finishBtn: 'Färdig',
      ap: {
        authTitle: 'Autentisering krävs',
        authDescription: 'Ange lösenordet AP för att fortsätta',
        authFailed: 'Ogiltigt AP lösenord',
        passPlaceholder: 'AP lösenord',
        verifyBtn: 'Verifiera'
      }
    },
    screen: {
      scale: 'Skala',
      title: 'Skärm',
      video: 'Videoläge',
      videoDirectTips: 'Aktivera HTTPS i "Inställningar > Enhet" för att använda detta läge',
      resolution: 'Upplösning',
      auto: 'Automatisk',
      autoTips:
        'Skärmtear eller musförskjutning kan förekomma vid vissa upplösningar. Överväg att justera fjärrvärdens upplösning eller inaktivera automatiskt läge.',
      fps: 'FPS',
      customizeFps: 'Anpassa',
      quality: 'Kvalitet',
      qualityLossless: 'Förlustfri',
      qualityHigh: 'Hög',
      qualityMedium: 'Medel',
      qualityLow: 'Låg',
      frameDetect: 'Ramdetection',
      frameDetectTip:
        'Beräkna skillnaden mellan ramar. Sluta skicka videoström när inga förändringar upptäcks på fjärrvärdens skärm.',
      resetHdmi: 'Återställ HDMI',
      mixedH264: {
        title: 'H.264-strömningskonflikt',
        description:
          'H.264 Direct och H.264 WebRTC används samtidigt. Detta kan orsaka skärmrivningar eller skadad video. Använd endast ett H.264-läge.'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC-anslutningen misslyckades',
        description: 'Kontrollera nätverksanslutningen eller byt videoläge.'
      },
      captureStatus: {
        hdmiError: 'HDMI-skärmfel',
        unsupportedResolution: 'Den aktuella upplösningen stöds inte',
        retrieving: 'Hämtar skärmen...',
        changingResolution: 'Byter upplösning...',
        updateFailed: 'Skärmen kan inte uppdateras just nu',
        videoError: 'Videovisningsfel',
        noHdmi: 'Ingen HDMI-signal upptäcktes',
        unavailable: 'Skärmen kan inte visas just nu'
      }
    },
    keyboard: {
      title: 'Tangentbord',
      paste: 'Klistra in',
      tips: 'Endast standardbokstäver och symboler på tangentbordet stöds',
      placeholder: 'Ange text',
      submit: 'Skicka',
      virtual: 'Tangentbord',
      readClipboard: 'Läs från Urklipp',
      clipboardPermissionDenied:
        'Behörighet till Urklipp nekad. Vänligen tillåt åtkomst till Urklipp i din webbläsare.',
      clipboardReadError: 'Misslyckades med att läsa Urklipp',
      dropdownEnglish: 'Engelska',
      dropdownGerman: 'Tyska',
      dropdownFrench: 'Franska',
      dropdownRussian: 'ryska',
      shortcut: {
        title: 'Genvägar',
        custom: 'Anpassad',
        capture: 'Klicka här för att fånga genväg',
        clear: 'Rensa',
        save: 'Spara',
        captureTips:
          'Att fånga systemtangenter (som Windows-tangenten) kräver helskärmsbehörighet.',
        enterFullScreen: 'Växla helskärmsläge.'
      },
      leaderKey: {
        title: 'Leader-tangent',
        desc: 'Gå förbi webbläsarbegränsningar och skicka systemgenvägar direkt till fjärrvärden.',
        howToUse: 'Hur man använder',
        simultaneous: {
          title: 'Samtidigt läge',
          desc1: 'Håll ned Leader-tangenten och tryck sedan på genvägen.',
          desc2: 'Intuitivt, men kan komma i konflikt med systemgenvägar.'
        },
        sequential: {
          title: 'Sekventiellt läge',
          desc1:
            'Tryck på Leader-tangenten → tryck på genvägen i följd → tryck på Leader-tangenten igen.',
          desc2: 'Kräver fler steg, men undviker helt systemkonflikter.'
        },
        enable: 'Aktivera Leader-tangent',
        tip: 'När den tilldelas som Leader-tangent fungerar denna tangent endast som genvägsutlösare och förlorar sitt standardbeteende.',
        placeholder: 'Tryck på Leader-tangenten',
        shiftRight: 'Höger Shift',
        ctrlRight: 'Höger Ctrl',
        metaRight: 'Höger Win',
        submit: 'Skicka',
        recorder: {
          rec: 'REC',
          activate: 'Aktivera tangenter',
          input: 'Vänligen tryck på genvägen...'
        }
      }
    },
    mouse: {
      title: 'Mus',
      cursor: 'Markörstil',
      default: 'Standardmarkör',
      pointer: 'Pekmarkör',
      cell: 'Cellmarkör',
      text: 'Textmarkör',
      grab: 'Greppmarkör',
      hide: 'Dölj markör',
      mode: 'Musläge',
      absolute: 'Absolut läge',
      relative: 'Relativt läge',
      direction: 'Rullhjulsriktning',
      scrollUp: 'Scrolla uppåt',
      scrollDown: 'Scrolla ner',
      speed: 'Rullhjulshastighet',
      fast: 'Snabb',
      slow: 'Långsam',
      requestPointer: 'Använder relativt läge. Klicka på skrivbordet för att få muspekaren.',
      resetHid: 'Återställ HID',
      hidOnly: {
        title: 'Endast HID-läge',
        desc: 'Om din mus och ditt tangentbord slutar svara och återställning av HID inte hjälper, kan det bero på kompatibilitetsproblem mellan NanoKVM och enheten. Prova att aktivera Endast-HID-läge för bättre kompatibilitet.',
        tip1: 'Aktivering av Endast-HID-läge avmonterar den virtuella U-disken och nätverket',
        tip2: 'I Endast-HID-läge är avbildningsmontering inaktiverat',
        tip3: 'NanoKVM kommer automatiskt att starta om efter lägesbyte',
        enable: 'Aktivera Endast-HID-läge',
        disable: 'Inaktivera Endast-HID-läge'
      }
    },
    image: {
      title: 'Avbildningar',
      loading: 'Laddar...',
      empty: 'Inget hittades',
      mountMode: 'Monteringsläge',
      mountFailed: 'Montering misslyckades',
      mountDesc:
        'I vissa system måste den virtuella disken avmonteras på fjärrvärden innan avbildningen monteras.',
      unmountFailed: 'Avmontering misslyckades',
      unmountDesc:
        'I vissa system måste du manuellt mata ut från fjärrvärden innan du avmonterar avbildningen.',
      refresh: 'Uppdatera avbildningslistan',
      attention: 'Observera',
      deleteConfirm: 'Är du säker på att du vill ta bort denna avbildning?',
      okBtn: 'Ja',
      cancelBtn: 'Nej',
      tips: {
        title: 'Hur man laddar upp',
        usb1: 'Anslut NanoKVM till din dator via USB.',
        usb2: 'Säkerställ att den virtuella disken är monterad (Inställningar - Virtuell Disk).',
        usb3: 'Öppna den virtuella disken på din dator och kopiera avbildningsfilen till rotkatalogen.',
        scp1: 'Säkerställ att NanoKVM och din dator är på samma lokala nätverk.',
        scp2: 'Öppna en terminal på din dator och använd SCP-kommandot för att ladda upp avbildningen till /data på NanoKVM.',
        scp3: 'Exempel: scp din-avbildningssökväg root@din-nanokvm-ip:/data',
        tfCard: 'TF-kort',
        tf1: 'Denna metod stöds på Linux-system',
        tf2: 'Ta ut TF-kortet från NanoKVM (för FULL-versionen, öppna chassit först).',
        tf3: 'Sätt in TF-kortet i en kortläsare och anslut till din dator.',
        tf4: 'Kopiera avbildningsfilen till /data på TF-kortet.',
        tf5: 'Sätt in TF-kortet i NanoKVM.'
      }
    },
    script: {
      title: 'Skript',
      upload: 'Ladda upp',
      run: 'Kör',
      runBackground: 'Kör i bakgrunden',
      runFailed: 'Körning misslyckades',
      attention: 'Observera',
      delDesc: 'Är du säker på att du vill ta bort denna fil?',
      confirm: 'Ja',
      cancel: 'Nej',
      delete: 'Ta bort',
      close: 'Stäng'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'NanoKVM Terminal',
      serial: 'Serieport-terminal',
      serialPort: 'Serieport',
      serialPortPlaceholder: 'Ange serieport',
      baudrate: 'Baudhastighet',
      parity: 'Paritet',
      parityNone: 'Ingen',
      parityEven: 'Jämn',
      parityOdd: 'Udda',
      flowControl: 'Flödeskontroll',
      flowControlNone: 'Ingen',
      flowControlSoft: 'Programvara',
      flowControlHard: 'Hårdvara',
      dataBits: 'Databitar',
      stopBits: 'Stoppbitar',
      confirm: 'Ok'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Skickar kommando...',
      sent: 'Kommando skickat',
      input: 'Ange MAC-adress',
      ok: 'Ok'
    },
    download: {
      title: 'Avbildningshämtare',
      input: 'Ange en fjärravbildnings-URL',
      ok: 'Ok',
      disabled: '/data partitionen är skrivskyddad, kan inte hämta avbildning',
      uploadbox: 'Släpp filen här eller klicka för att välja',
      inputfile: 'Vänligen ange bildfilen',
      NoISO: 'Ingen ISO',
      sha256: 'SHA-256 (valfrie)',
      sha256Placeholder: 'Skriv inn en SHA-256-kontrollsum på 64 tegn',
      invalidSHA256: 'SHA-256 må være en heksadesimal streng på 64 tegn',
      failed: 'Nedlasting mislyktes',
      success: 'Nedlasting fullført',
      checksumFailed: 'Nedlasting mislyktes: SHA-256-verifisering mislyktes',
      cancel: 'Avbryt',
      cancelFailed: 'Kunne ikke avbryte nedlastingen'
    },
    power: {
      title: 'Ström',
      showConfirm: 'Bekräftelse',
      showConfirmTip: 'Strömätgärder kräver extra bekräftelse',
      reset: 'Starta om',
      power: 'Ström',
      powerShort: 'Ström (kort tryck)',
      powerLong: 'Ström (långt tryck)',
      resetConfirm: 'Utföra omstart?',
      powerConfirm: 'Utföra strömåtgärd?',
      okBtn: 'Ja',
      cancelBtn: 'Nej'
    },
    devices: {
      title: 'Enheter',
      stale: 'Enheternas livestatus är inte tillgänglig. Återansluter.',
      empty:
        'Inga kamera- eller mikrofonplatser är konfigurerade. Lägg till en under Inställningar, Enhet.',
      available: 'Tillgänglig',
      waiting: 'Värden väntar på en källa',
      hostOpen: 'Värden öppen',
      hostIdle: 'Värden inaktiv',
      sending: 'Sänder från den här webbläsaren',
      black: 'Svart video',
      silence: 'Digital tystnad',
      resuming: 'Väntar på att återupptas',
      stop: 'Sluta dela',
      disconnect: 'Koppla från',
      takeover: 'Ta över',
      refused: 'Används av {{owner}} från {{source}}',
      connectedSources_one: '{{count}} ansluten källa',
      connectedSources_other: '{{count}} anslutna källor',
      connectedSources: '{{count}} anslutna källor',
      connection: {
        connecting: 'Ansluter',
        connected: 'Direkt',
        disconnected: 'Återansluter'
      },
      share: {
        camera: 'Dela kamera',
        microphone: 'Dela mikrofon',
        usbDevice: 'Dela USB'
      },
      permission: {
        denied: 'Blockerat i webbläsarens webbplatsinställningar',
        prompt: 'Webbläsaren kommer att fråga om åtkomst',
        insecure:
          'Den här sidan levereras inte över HTTPS, så webbläsaren blockerar den här enheten. Aktivera HTTPS under Inställningar, Nätverk.'
      },
      capture: {
        unsupported: 'Den här webbläsaren kan inte spela in ljud eller video',
        camera: 'Den här webbläsaren kan inte koda kamerabilder',
        microphone: 'Den här webbläsaren kan inte bearbeta mikrofonljud'
      },
      mic: {
        mute: 'Stäng av ljudet',
        unmute: 'Slå på ljudet'
      },
      revoked: {
        released: 'Delningen stoppades',
        lease_expired: 'Leaset gick ut innan den här webbläsaren kom tillbaka',
        admin_disconnect: 'En administratör kopplade från alla källor',
        slot_removed: 'Platsen togs bort',
        slot_changed: 'Platsen konfigurerades om',
        taken_over: 'En administratör tog över den här platsen'
      },
      usb: {
        surrendered: 'USB-passthrough håller tangentbordet och musen',
        surrenderedDesc:
          'Fjärrvärden ser den importerade enheten i stället för NanoKVM:s tangentbord, mus och virtuella media. De kommer tillbaka när sessionen stoppas.',
        unsupported: 'WebUSB kräver en Chromium-webbläsare',
        insecure:
          'Den här sidan levereras inte över HTTPS, så webbläsaren håller tillbaka WebUSB. Aktivera HTTPS under Inställningar, Nätverk.',
        session: 'Vidarebefordrar {{device}} ({{mode}})',
        idle: 'Ingen passthrough-session',
        mode: {
          hybrid: 'hybrid',
          exact: 'exakt'
        }
      }
    },
    settings: {
      title: 'Inställningar',
      display: {
        title: 'Skärm',
        loading: 'Laddar...',
        active: 'Aktiv EDID',
        activeUnknown:
          'NanoKVM har inte skrivit någon EDID sedan den startade, så det är okänt vilken skärm värden ser.',
        appliedAt: 'Tillämpad {{time}}',
        download: 'Ladda ner',
        downloadBackup: 'Ladda ner föregående',
        preset: 'Skärmförval',
        presetPlaceholder: 'Välj en skärm',
        upload: 'Ladda upp',
        selected: 'Vald EDID',
        errors: 'Fel',
        warnings: 'Varningar',
        info: 'Information',
        unknownMonitor: 'Okänd skärm',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Ljud',
        audioNo: 'Inget ljud',
        extensionBlocks: 'Tilläggsblock: {{blocks}}',
        apply: 'Tillämpa',
        applyTitle: 'Vill du tillämpa den här EDID:n?',
        before: 'Nuvarande',
        after: 'Ny',
        hdmiNotice:
          'Videoinspelningen stannar medan EDID skrivs och startar sedan av sig själv igen.',
        powerCycleNotice:
          'Enheten måste kopplas bort fysiskt från strömmen och anslutas igen innan den nya EDID:n börjar gälla.',
        powerCycleUnverified:
          'Skrivningen verifierades inte, så videochippet behåller det som ligger i det nu tills enheten kopplas bort fysiskt från strömmen och ansluts igen.',
        applied: 'EDID tillämpad och verifierad.',
        applyFailed: 'Det gick inte att tillämpa EDID.',
        busy: 'Videokretsen var upptagen. Försök igen.',
        unsupported: 'Den här enheten stöder inte att EDID ändras.',
        toolMissing: 'EDID-verktyget saknas i den här fastvaran.',
        noAudio: 'Den här EDID:n anger inget ljud, så värden kan sluta skicka ljud.',
        oldVersion: 'Den här EDID:n använder en äldre version än 1.4.',
        interlaced: 'Den föredragna upplösningen är sammanflätad.',
        tooLarge:
          'Den föredragna upplösningen är större än 1920x1080 vid 60 Hz, vilket är mer än NanoKVM kan fånga.',
        recovery: 'Återställning',
        recoveryNeeded:
          'Den senaste skrivningen verifierades inte, så EDID-området i videochippet är i ett okänt tillstånd. Återställ fabriks-EDID för att få ett känt tillstånd igen.',
        recoveryDesc:
          'Skriv tillbaka ett känt EDID till videochippet om det du tillämpade lämnade värden utan bild.',
        restoreFactory: 'Återställ fabriks-EDID',
        restoreBackup: 'Återställ föregående EDID',
        restoreTitle: 'Vill du återställa det här EDID:t?',
        restoreFactoryTarget: 'Fabriks-EDID:t som NanoKVM levereras med.',
        restoreBackupTarget: 'Den senaste säkerhetskopian, tillämpad {{time}}.',
        restoreNotice:
          'En återställning skrivs på samma sätt som en tillämpning och har samma konsekvenser.',
        restored: 'EDID återställt och verifierat.',
        restoreFailed: 'Det gick inte att återställa EDID.',
        mismatchTitle: 'Skrivet och återläst',
        mismatchWritten: 'Skrivet',
        mismatchRead: 'Återläst',
        restoreOkBtn: 'Återställ',
        hardware: 'Identifierad hårdvara: {{hardware}}',
        hardwareUnknown: 'Okänd',
        confirmWord: 'TILLÄMPA',
        confirmPrompt: 'Skriv {{word}} för att aktivera tillämpa-knappen.',
        okBtn: 'Tillämpa',
        cancelBtn: 'Avbryt'
      },
      presentation: {
        title: 'USB-presentation',
        loading: 'Laddar...',
        current: 'Nuvarande USB-presentation',
        noProfile: 'Ingen profil tillämpad',
        linked: 'Länkade funktioner',
        hostState: 'Värdens USB',
        hostUnbound: 'Styrkretsen är inte bunden',
        hdmiState: 'HDMI-ingång',
        hdmiSignal: 'Signal finns',
        hdmiUnreported: 'Ingen rapport om inspelning ännu',
        endpoints: 'Endpoints',
        fifos: 'FIFO-platser',
        pending: 'Väntande ändringar',
        pendingEdits: 'Osparade identitetsändringar',
        pendingProfile: '{{profile}} är vald men inte tillämpad',
        pendingNone: 'Inga',
        lastApply: 'Senaste tillämpning',
        applyFailed: 'Misslyckades på {{profile}} den {{time}}',
        applyClean: 'Inget fel registrerat',
        lastKnownGood: 'Senast kända fungerande',
        rollbackTarget: 'Mål för återställning',
        rollbackNone: 'Inget',
        powerCyclePending:
          'Styrkretsen togs från värden. Stäng av och slå på den anslutna datorn igen för att få tillbaka enheten.',
        rollback: 'Återställ',
        rollbackTitle: 'Återställa till {{profile}}?',
        rollbackDesc: 'Gadgeten räknas upp på nytt; USB-funktioner faller bort en kort stund.',
        profile: 'USB-profil',
        builtIn: 'inbyggd',
        descriptors: 'deskriptorer',
        imported: 'importerad',
        clone: 'Klona',
        cloneTitle: 'Klona den här profilen',
        cloneToEdit:
          'Inbyggda profiler förblir skrivskyddade. Klona den här profilen för att redigera dess identitet.',
        profileName: 'Profilnamn',
        profileNameHint: 'Små bokstäver, siffror, punkter, understreck och bindestreck.',
        import: 'Importera paket',
        export: 'Exportera paket',
        delete: 'Ta bort',
        deleteTitle: 'Ta bort den här profilen?',
        deleteDesc: 'Ta bort {{profile}} från NanoKVM.',
        identity: 'USB-identitet',
        preset: 'Förvald identitet',
        presetPlaceholder: 'Kopiera identitet från en känd enhet',
        presetHint:
          'Ett förval fyller i Vendor ID, Product ID och de två namnfälten. Det för inte med sig några deskriptorer.',
        presetSource: 'Identitet så som den är noterad i {{source}}',
        vendorId: 'Vendor ID',
        foreignVendor: 'Det här Vendor ID:t tillhör en annan tillverkare',
        productId: 'Product ID',
        bcdUSB: 'USB-version',
        bcdDevice: 'Enhetsversion',
        manufacturer: 'Tillverkare',
        product: 'Produkt',
        serial: 'Serienummer',
        configuration: 'Konfigurationssträng',
        hidLayout: 'HID-enheter',
        hidRoleKeyboard: 'Tangentbord',
        hidRoleRelative: 'Mus (relativ)',
        hidRoleAbsolute: 'Pekare (absolut)',
        hidOff: 'Finns inte',
        hidInterface: 'Gränssnitt {{index}}',
        hidBootKeyboardShared:
          'Tangentbordet delar ett gränssnitt och erbjuder därför inte längre någon rapport i boot-protokoll. Vissa BIOS- och UEFI-uppsättningar kommer inte att se det.',
        functions: 'Funktioner',
        descriptorAssets: 'Sparade deskriptorfiler: {{count}}',
        endpointUse:
          'IN {{inUse}} använda, {{inFree}} lediga; OUT {{outUse}} använda, {{outFree}} lediga',
        apply: 'Tillämpa',
        applyTitle: 'Vill du tillämpa den här USB-profilen?',
        applyDesc: 'NanoKVM kommer att presentera {{profile}} för den anslutna datorn.',
        reconnect:
          'Tangentbord, mus och andra USB-funktioner kopplas bort en kort stund medan gadgeten binds om.',
        applyLinks: 'Länkar: {{functions}}',
        applyRemoves: 'Tar bort: {{functions}}',
        applyNoHid:
          'Ingen HID-funktion återstår efter den här tillämpningen. Tangentbord och mus slutar fungera.',
        applyRollback: 'En misslyckad tillämpning går tillbaka till {{profile}}.',
        recoveryPowerCycle:
          'Ingen HID överlever den här tillämpningen, så en värd som slutar svara går bara att rädda genom att stänga av och slå på strömmen.',
        recoveryReboot:
          'Ett gränssnitt försvinner från den sammansatta enheten; värden kan behöva startas om för att binda resten på nytt.',
        recoveryHdmiReset:
          'En videofunktion byggs upp på nytt, så inspelningskedjan bakom den nollställs.',
        recoveryReconnect:
          'Värden räknar upp enheten på nytt; USB-funktioner faller bort en kort stund.',
        cancel: 'Avbryt',
        noFunctions: 'Inga länkade funktioner',
        loadFailed: 'Det gick inte att läsa in presentationsprofilerna',
        operationFailed: 'Presentationsåtgärden misslyckades'
      },
      passthrough: {
        title: 'USB-genomsläpp',
        loading: 'Laddar...',
        mode: 'Läge',
        hybrid: 'Hybrid',
        exact: 'Exakt',
        hybridDesc: 'Behåller boot-tangentbordet och den relativa musen, för kompatibla enheter.',
        exactDesc: 'Ersätter varje USB-funktion i NanoKVM med den importerade enheten.',
        hybridWarning: 'Hybrid håller tangentbordet och den relativa musen kvar',
        hybridWarningDesc:
          'Lagring, USB-nätverk och den absoluta pekaren kopplas bort medan den importerade funktionen är aktiv.',
        hidWarning:
          'Att starta genomsläpp lämnar ifrån sig tangentbordet, musen och virtuella media',
        hidWarningDesc:
          'NanoKVM har bara en USB-enhetsstyrenhet och proxyn behöver hela den. Medan en session pågår ser därför fjärrvärden den vidarekopplade enheten i stället för NanoKVM:s tangentbord, mus och virtuella media. De kommer tillbaka av sig själva i samma stund som sessionen stoppas. Detta webbgränssnitt påverkas inte, så du kan alltid stoppa en session från den här sidan.',
        hidWarningSafeDesc:
          'NanoKVM har bara en USB-enhetsstyrenhet och proxyn behöver hela den. Medan en session pågår ser därför fjärrvärden den vidarekopplade enheten i stället för NanoKVM:s tangentbord, mus och virtuella media. De kommer tillbaka när sessionen stoppas.',
        isoLabel: 'Tillåt isokrona överföringar',
        isoHint:
          'Släpper igenom webbkameror, mikrofoner och andra strömmande enheter. Ingen har mätt vad den här maskinvaran orkar.',
        isoWarning:
          'Isokron strömning är oprövad här och kan hålla kvar tangentbordet och musen tills du stoppar sessionen',
        info: {
          title: 'Info',
          hybrid:
            'Hybridläget håller tangentbordet och den relativa musen kvar. Lagring, USB-nätverk och den absoluta pekaren kopplas bort medan den importerade enheten är aktiv.',
          exact:
            'Exakt läge ersätter varje USB-funktion i NanoKVM med den importerade enheten. Tangentbordet, musen och de virtuella medierna kommer tillbaka av sig själva när sessionen stoppas.',
          udc: 'NanoKVM har bara en USB-enhetsstyrenhet och proxyn behöver hela den — det är därför funktionerna ovan försvinner så länge en session pågår.',
          web: 'Detta webbgränssnitt påverkas inte, så du kan alltid stoppa en session från den här sidan.',
          network:
            'Starta genomsläpp över Ethernet eller Wi-Fi. Att starta det från NanoKVM:s USB-nätverk avvisas, eftersom den anslutningen skulle försvinna.',
          iso: 'Webbkameror, mikrofoner och andra isokrona enheter avvisas tills du tillåter isokrona överföringar. Den vägen fungerar men har aldrig mätts på den här maskinvaran, så betrakta genomströmningen som okänd.',
          camera:
            'Webbläsarens kamera och mikrofon under Enheter är fortfarande det beprövade sättet att ge värden en.'
        },
        session: 'Session',
        activeDesc: 'En enhet är importerad och proxyn håller USB-styrenheten.',
        inactiveDesc:
          'Ingen session pågår. Tangentbord, mus och virtuella media fungerar som vanligt.',
        device: 'Enhet',
        busId: 'Buss-ID',
        speed: 'Hastighet',
        exporter: 'Exportör',
        local: 'Importerad som',
        localValue: 'Buss {{bus}}, adress {{address}}',
        udc: 'USB-styrenhet',
        pid: 'Proxyns PID',
        startedAt: 'Startad',
        isoDevice:
          'Den här enheten strömmar över isokrona slutpunkter, vilket aldrig har mätts på den här maskinvaran',
        exporterLabel: 'Exportörens adress',
        exporterHint:
          'Värden och porten som NanoKVM ringer upp. Via tunneln nedan är det {{exporter}}.',
        busIdLabel: 'Buss-ID på din egen dator',
        busIdHint: 'Det busid som usbip list -l skriver ut för enheten, till exempel {{example}}.',
        start: 'Starta genomsläpp',
        stop: 'Stoppa genomsläpp',
        startTitle: 'Vill du starta USB-genomsläpp?',
        startDevice: 'NanoKVM importerar {{busId}} från {{exporter}}.',
        startHid:
          'USB-tangentbordet, musen och virtuella media slutar fungera så länge sessionen pågår och startar av sig själva igen när du stoppar den.',
        startIso:
          'Webbkameror och andra isokrona enheter kräver att du slår på den isokrona brytaren innan du startar.',
        startWeb:
          'Detta webbgränssnitt fortsätter att fungera, så du kan stoppa sessionen från den här sidan när som helst.',
        startNetwork:
          'Använd den här sidan över Ethernet eller Wi-Fi. Att starta från NanoKVM:s USB-nätverk avvisas eftersom den anslutningen skulle försvinna.',
        okBtn: 'Starta',
        cancelBtn: 'Avbryt',
        instructions: 'På din egen dator',
        instructionsDesc:
          'Det finns medvetet ingen klientagent att installera. Kör dessa vanliga usbip-kommandon på datorn som enheten sitter i.',
        copyFailed: 'Kopieringen misslyckades. Kopiera kommandot manuellt.',
        copyInsecure:
          'Den här sidan levereras inte över HTTPS, så webbläsaren blockerar kopiering. Kopiera kommandot manuellt, eller aktivera HTTPS under Inställningar, Nätverk.',
        directNote:
          'Utan tunnel måste usbipd vara nåbar på ditt nätverk och exportöradressen ovan måste peka på den. usbip skickar enheten okrypterad, så tunneln är att föredra.',
        steps: {
          modprobe: {
            title: 'Ladda drivrutinen på exportsidan',
            desc: 'usbip-host är det som låter kärnan lämna ifrån sig en lokal enhet. Den laddas inte som standard.'
          },
          list: {
            title: 'Hitta enheten',
            desc: 'Skriver ut varje lokal enhet med sitt busid och sitt tillverkare:produkt-par. Notera busid för den du vill använda.'
          },
          bind: {
            title: 'Bind den till usbip',
            desc: 'Tar enheten från dess vanliga drivrutin, så den slutar fungera på den här datorn tills du löser bindningen.'
          },
          serve: {
            title: 'Dela ut den',
            desc: 'usbipd stannar i förgrunden och väntar på att NanoKVM ska importera enheten.',
            notice:
              'Vanliga usbipd har ingen inställning för lyssnaradress och lyssnar på alla gränssnitt. Håll port {{port}} stängd i brandväggen och låt bara tunneln nedan nå den.'
          },
          tunnel: {
            title: 'Rikta den mot NanoKVM',
            desc: 'En omvänd SSH-tunnel: port {{port}} på NanoKVM:s egen loopback blir usbipd på den här datorn. Låt den vara igång under hela sessionen.'
          },
          exporter: {
            title: 'Använd detta som exportör',
            desc: 'Skriv in detta i exportörfältet ovan, ange buss-ID:t och starta sessionen.'
          },
          unbind: {
            title: 'Lämna tillbaka enheten',
            desc: 'När sessionen har stoppats lämnar detta tillbaka enheten till dess vanliga drivrutin på den här datorn.'
          }
        }
      },
      mcp: {
        title: 'MCP-tjänst',
        service: 'MCP-fjärrstyrning',
        serviceDesc:
          'Tillåt betrodda MCP-klienter att styra tangentbord och mus och ta skärmbilder',
        securityWarning:
          'Alla som har denna API-nyckel kan styra fjärrvärden och se dess skärm. Använd HTTPS och aktivera tjänsten endast i betrodda nätverk.',
        endpoint: 'Slutpunkt',
        apiKey: 'API-nyckel',
        regenerateConfirmTitle: 'Generera om MCP API-nyckeln?',
        regenerateConfirmDesc: 'Den aktuella nyckeln slutar omedelbart att fungera.',
        enableConfirmTitle: 'Aktivera extern MCP-styrning?',
        enableConfirmDesc:
          'Om MCP aktiveras stoppas PicoClaw och alla aktiva PicoClaw-sessioner stängs.',
        failed: 'MCP-åtgärden misslyckades',
        copyFailed: 'Kopiering misslyckades. Kopiera manuellt.',
        copyInsecure:
          'Den här sidan levereras inte över HTTPS, så webbläsaren blockerar kopiering. Kopiera manuellt, eller aktivera HTTPS under Inställningar, Nätverk.',
        okBtn: 'Bekräfta',
        cancelBtn: 'Avbryt'
      },
      about: {
        title: 'Om NanoKVM',
        information: 'Information',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Applikationsversion',
        applicationTip: 'NanoKVM webbapplikationsversion',
        image: 'Systemversion',
        imageTip: 'NanoKVM systemavbildningsversion',
        deviceKey: 'Enhetsnyckel',
        community: 'Community',
        hostname: 'Värdnamn',
        hostnameUpdated: 'Värdnamn uppdaterat. Starta om för att tillämpa.',
        ipType: {
          Wired: 'Trådbundet',
          Wireless: 'Trådlöst',
          Other: 'Annat'
        }
      },
      appearance: {
        title: 'Utseende',
        display: 'Visning',
        language: 'Språk',
        languageDesc: 'Välj språk för gränssnittet',
        webTitle: 'Webbtitel',
        webTitleDesc: 'Anpassa webbsidans titel',
        favicon: 'Favicon',
        faviconDesc: 'Anpassa ikonen i webbläsarfliken',
        faviconPlaceholder: 'Bild-URL',
        faviconUpload: 'Ladda upp',
        faviconReset: 'Återställ',
        faviconCustom: 'Egen ikon',
        faviconBoot: 'Ikon från /boot/logo.ico',
        faviconDefault: 'Standardikon',
        faviconOverridesBoot: 'Åsidosätter /boot/logo.ico',
        faviconErrUrl: 'Ange en http:// eller https:// bildadress',
        faviconErrFetch: 'Enheten kunde inte hämta bilden',
        faviconErrLarge: 'Bilden är för stor. Gränsen är 256 KB',
        faviconErrType: 'Bildformatet stöds inte. Använd .ico, .png, .svg, .gif eller .jpg',
        faviconErrSave: 'Kunde inte spara ikonen',
        menuBar: {
          title: 'Menyrad',
          mode: 'Visningsläge',
          modeDesc: 'Visa menyraden på skärmen',
          modeOff: 'Av',
          modeAuto: 'Dölj automatiskt',
          modeAlways: 'Alltid synlig',
          keyboardLedStatus: 'Indikatorer för tangentbordslås',
          keyboardLedStatusDesc:
            'Visa Num Lock-, Caps Lock- och Scroll Lock-status för fjärrdatorn',
          icons: 'Undermenyikoner',
          iconsDesc: 'Visa undermenyikoner i menyraden'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Status för lås på fjärrtangentbord',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'På',
        off: 'Av',
        unknown: 'Okänd'
      },
      device: {
        title: 'Enhet',
        oled: {
          title: 'OLED',
          description: 'Stäng av OLED-skärmen efter',
          0: 'Aldrig',
          15: '15 sek',
          30: '30 sek',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 timme'
        },
        ssh: {
          description: 'Aktivera SSH-fjärråtkomst',
          tip: 'Ställ in ett starkt lösenord innan du aktiverar (Konto - Byt lösenord)'
        },
        advanced: 'Avancerade inställningar',
        swap: {
          title: 'Swap',
          disable: 'Inaktivera',
          description: 'Ange swap-filens storlek',
          tip: 'Aktivering av denna funktion kan förkorta livslängden på ditt SD-kort!'
        },
        mouseJiggler: {
          title: 'Musvickare',
          description: 'Förhindra att fjärrvärden går i viloläge',
          disable: 'Inaktivera',
          absolute: 'Absolut läge',
          relative: 'Relativt läge'
        },
        mdns: {
          description: 'Aktivera mDNS-upptäckningstjänst',
          tip: 'Stäng av om det inte behövs'
        },
        hdmi: {
          description: 'Aktivera HDMI/monitorutgång',
          idleTimeoutTitle: 'Tidsgräns för inaktiv inspelning',
          idleTimeoutDescription:
            'Stoppa HDMI-inspelning efter att det inte har funnits aktiva tittare i',
          minutes: 'min'
        },
        autostart: {
          title: 'Autostart skriptinställningar',
          description: 'Hantera skript som körs automatiskt vid systemstart',
          new: 'Nytt',
          deleteConfirm: 'Är du säker på att du vill ta bort denna fil?',
          yes: 'Ja',
          no: 'Nej',
          scriptName: 'Autostart skriptnamn',
          scriptContent: 'Autostart skriptinnehåll',
          settings: 'Inställningar'
        },
        hidOnly: 'Endast-HID-läge',
        hidOnlyDesc: 'Sluta emulera virtuella enheter, behåll bara grundläggande HID kontroll',
        disk: 'Virtuell disk',
        diskDesc: 'Montera virtuell U-disk på fjärrvärden',
        rebindNotice:
          'Att slå om den här brytaren räknar upp USB-enheten på nytt, så målet förlorar kort sina virtuella enheter och sitt USB-nätverk.',
        media: {
          title: 'Platser för kamera och mikrofon',
          desc: 'Ange vilka medieenheter webbläsare får fylla. Endpoint-budgeten kontrolleras när USB-profilen tillämpas. Att spara räknar upp enheten på nytt och kopplar från anslutna webbläsare.',
          cameras: 'Kameror',
          microphones: 'Mikrofoner',
          name: 'Namn',
          namePlaceholder: 'Visas på måldatorn',
          addCamera: 'Lägg till kamera',
          addMicrophone: 'Lägg till mikrofon',
          remove: 'Ta bort',
          cameraDefault: 'Kamera {{index}}',
          microphoneDefault: 'Mikrofon {{index}}',
          nameRequired: 'Varje plats behöver ett namn.',
          budgetHint:
            'De sex USB IN-slutpunkterna är en fast hårdvarugräns. Samla tangentbord, mus och absolut pekdon på ett enda HID-gränssnitt under USB-presentation, eller stäng av den virtuella disken här eller USB-nätverkskortet under Nätverk.',
          unsupported:
            'Den här kärnan kan inte namnge medieenheter, så värdar visar standardnamnet.',
          save: 'Spara platser',
          disconnect: 'Koppla från',
          disconnectAll: 'Koppla från alla källor',
          limit: 'Platser för kamera och mikrofon får sammanlagt vara högst åtta.',
          failed: 'Medieplatserna kunde inte uppdateras.'
        },
        reboot: 'Starta om',
        rebootDesc: 'Är du säker på att du vill starta om NanoKVM?',
        okBtn: 'Ja',
        cancelBtn: 'Nej'
      },
      network: {
        title: 'Nätverk',
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurera Wi-Fi',
          apMode: 'AP-läge är aktiverat, anslut till Wi-Fi genom att skanna QR-koden',
          connect: 'Anslut Wi-Fi',
          connectDesc1: 'Ange nätverkets SSID och lösenord',
          connectDesc2: 'Ange lösenordet för att ansluta till detta nätverk',
          disconnect: 'Är du säker på att du vill koppla från nätverket?',
          failed: 'Anslutningen misslyckades, försök igen.',
          ssid: 'Namn',
          password: 'Lösenord',
          joinBtn: 'Anslut',
          confirmBtn: 'OK',
          cancelBtn: 'Avbryt'
        },
        tls: {
          description: 'Aktivera HTTPS-protokoll',
          tip: 'Observera: Användning av HTTPS kan öka fördröjningen, särskilt med MJPEG-läge.'
        },
        usb: {
          title: 'USB-nätverkskort',
          desc: 'Ger den styrda datorn ett nätverkskort över USB',
          type: 'Korttyp',
          typeDesc: 'NCM för moderna system, RNDIS för äldre Windows'
        },
        bridge: {
          title: 'Kortet är anslutet till',
          lan: 'Ditt nätverk',
          kvmOnly: 'Bara NanoKVM',
          lanDesc:
            'Datorn kommer in på ditt nätverk genom NanoKVM:s Ethernet-port och får en egen adress från routern.',
          kvmOnlyDesc: 'Datorn får sin adress från NanoKVM och når NanoKVM, men inget bortom det.',
          loading: 'Laddar...',
          state: 'Status',
          states: {
            disabled: 'Bara NanoKVM',
            enabled: 'Ditt nätverk',
            rolledBack: 'Återställd',
            failed: 'Misslyckades',
            pending: 'Pågår'
          },
          uplink: 'Upplänk',
          ports: 'Portar',
          up: 'aktiv',
          down: 'inaktiv',
          noLink: 'ingen länk',
          enableTitle: 'Ansluta datorn till ditt nätverk?',
          disableTitle: 'Begränsa datorn till bara NanoKVM?',
          reconnect: 'Hanteringsanslutningen bryts kort och återansluter medan adressen flyttas.',
          rollback:
            'Om verifieringen misslyckas återställs den tidigare konfigurationen automatiskt.',
          enableBtn: 'Anslut till mitt nätverk',
          disableBtn: 'Bara NanoKVM',
          cancelBtn: 'Avbryt',
          interrupted:
            'Anslutningen bröts under tillämpningen. Kontrollerar nuvarande status igen.',
          pendingNotice:
            'En ändring av bryggan pågår fortfarande eller avbröts innan den blev klar.',
          revert: 'Återställ tidigare konfiguration',
          rolledBackNotice:
            'Den senaste ändringen återkallades och den tidigare konfigurationen återställdes.',
          verifyFailed: 'Verifieringen misslyckades: {{gates}}',
          gates: {
            address: 'adress',
            gateway: 'gateway',
            inbound: 'inkommande anslutning'
          },
          inboundWeak:
            'Kontrollen av inkommande trafik gick igenom bara för att NanoKVM anslöt till sig själv. Det visar att webbtjänsten lyssnar och nås lokalt, inte att en begäran från nätverket kommer fram.',
          noCarrier:
            'Ingen länk på {{port}}. Bryggan har ingen väg ut på nätverket förrän en kabel ansluts.',
          loop: 'Routern lärs in även på {{port}}, så den porten är en andra väg till samma nätverk. Spanning tree är avstängt, så inget här bryter slingan: koppla bort en av de två vägarna.',
          failedNotice:
            'Den senaste ändringen kunde inte ångras. NanoKVM kanske bara går att nå via Wi-Fi-accesspunkten eller en seriell konsol.'
        },
        dns: {
          title: 'DNS',
          description: 'Konfigurera DNS-servrar för NanoKVM',
          mode: 'Läge',
          dhcp: 'DHCP',
          manual: 'Manuell',
          add: 'Lägg till DNS',
          save: 'Spara',
          invalid: 'Ange en giltig IP-adress',
          noDhcp: 'Ingen DHCP-DNS är tillgänglig just nu',
          saved: 'DNS-inställningar sparade',
          saveFailed: 'Det gick inte att spara DNS-inställningar',
          unsaved: 'Osparade ändringar',
          maxServers: 'Maximalt {{count}} DNS-servrar tillåtna',
          dnsServers: 'DNS-servrar',
          dhcpServersDescription: 'DNS-servrar hämtas automatiskt från DHCP',
          manualServersDescription: 'DNS-servrar kan redigeras manuellt',
          networkDetails: 'Nätverksdetaljer',
          interface: 'Gränssnitt',
          ipAddress: 'IP-adress',
          subnetMask: 'Subnätmask',
          router: 'Router',
          none: 'Ingen'
        }
      },
      vnc: {
        title: 'VNC',
        server: 'VNC-server',
        description:
          'Låt vilken VNC-klient som helst se fjärrskärmen och använda tangentbord och mus, med inloggning via ditt NanoKVM-konto',
        port: 'Port',
        portDescription: 'Anslut till den här porten på NanoKVM-adressen'
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Minnesoptimering',
          tip: 'När minnesanvändningen överskrider gränsen utförs aggressivare skräpsamling för att frigöra minne. Rekommenderas att sättas till 75 MB om du använder Tailscale. Omstart krävs för att det ska gälla.'
        },
        swap: {
          title: 'Byt minne',
          tip: 'Om problemen kvarstår efter att du har aktiverat minnesoptimering, försök att aktivera utbyte av minne. Detta ställer in växlingsfilens storlek till 256MB som standard, vilket kan justeras i "Inställningar > Enhet".'
        },
        restart: 'Starta om Tailscale?',
        stop: 'Stoppa Tailscale?',
        stopDesc: 'Logga ut från Tailscale och inaktivera autostart vid uppstart.',
        loading: 'Laddar...',
        notInstall: 'Tailscale hittades inte! Installera först.',
        install: 'Installera',
        installing: 'Installerar',
        failed: 'Installationen misslyckades',
        retry: 'Uppdatera sidan och försök igen. Eller installera manuellt',
        download: 'Ladda ner',
        package: 'installationspaketet',
        unzip: 'och packa upp det',
        upTailscale: 'Ladda upp tailscale till NanoKVM-katalogen /usr/bin/',
        upTailscaled: 'Ladda upp tailscaled till NanoKVM-katalogen /usr/sbin/',
        refresh: 'Uppdatera sidan',
        notRunning: 'Tailscale körs inte. Starta den för att fortsätta.',
        run: 'Start',
        notLogin: 'Enheten är ännu inte bunden. Logga in och bind enheten till ditt konto.',
        urlPeriod: 'Denna URL är giltig i 10 minuter',
        login: 'Logga in',
        loginSuccess: 'Inloggning lyckades',
        enable: 'Aktivera Tailscale',
        deviceName: 'Enhetsnamn',
        deviceIP: 'Enhets-IP',
        account: 'Konto',
        logout: 'Logga ut',
        logoutDesc: 'Är du säker på att du vill logga ut?',
        uninstall: 'Avinstallera Tailscale',
        uninstallDesc: 'Är du säker på att du vill avinstallera Tailscale?',
        okBtn: 'Ja',
        cancelBtn: 'Nej'
      },
      wstunnel: {
        title: 'wstunnel'
      },
      newt: {
        title: 'Newt'
      },
      tunnel: {
        loading: 'Laddar...',
        notInstall: 'Inte installerad',
        notConfigured: 'Inte konfigurerad',
        stopped: 'Stoppad',
        running: 'Körs',
        connected: 'Ansluten',
        error: 'Fel',
        atBoot: 'startar vid uppstart',
        notAtBoot: 'startar inte vid uppstart',
        arguments: 'Argument',
        argumentsTip: 'Kommandoradsargument som skickas till tjänsten vid start.',
        env: 'Miljövariabler',
        envKey: 'Namn',
        envValue: 'Värde',
        envAdd: 'Lägg till variabel',
        envRemove: 'Ta bort',
        configured: 'Konfigurerad',
        save: 'Spara',
        saved: 'Konfigurationen har sparats',
        start: 'Starta',
        stop: 'Stoppa',
        restart: 'Starta om',
        logs: 'Logg',
        logsEmpty: 'Inga loggposter ännu',
        refresh: 'Uppdatera',
        binary: 'Binärfil',
        binaryShipped: 'Medföljer firmware',
        binaryCustom: 'Egen binärfil',
        binaryUpload: 'Ladda upp binärfil',
        binaryRevert: 'Återställ medföljande binärfil',
        binaryRevertDesc:
          'Ta bort den uppladdade binärfilen och återställa den som medföljer firmware?',
        serverWarning: 'En server utan begränsningar fungerar som en öppen proxy',
        noHealthSignal:
          'Tjänsten rapporterar ingen hälsostatus, så NanoKVM vet bara att processen körs, inte om tunneln är ansluten.',
        memoryWarning: 'Att köra flera fjärråtkomsttjänster samtidigt kan ta slut på minnet',
        resources: 'Resurser',
        memory: {
          title: 'Minnesgräns',
          description:
            'Begränsar newts Go-heap till {{limit}} MiB från nästa omstart. Dess egen gräns, inte Tailscales; avstängd gäller Gos standardvärde, och GOGC=50 tillämpas i båda fallen.',
          noRuntime:
            'wstunnel är Rust: ingen skräpsamlare och ingen heap-gräns att sätta, och dess arbetstrådar följer redan enhetens enda CPU.',
          notApplicable: 'Ej tillämpligt'
        },
        swap: {
          title: 'Växlingsfil',
          description:
            'Lägger till en växlingsfil på 256 MB på SD-kortet. Den gäller hela systemet: samma växling betjänar Tailscale, KVM-servern och allt annat på enheten.'
        },
        okBtn: 'Ja',
        cancelBtn: 'Nej'
      },
      update: {
        title: 'Sök efter uppdateringar',
        queryFailed: 'Kunde inte hämta version',
        updateFailed: 'Uppdatering misslyckades. Försök igen.',
        isLatest: 'Du har redan den senaste versionen.',
        rebooting:
          'Den nya kärnan installeras och enheten startar om. Det kan ta några minuter; stäng inte av strömmen.',
        kernelUpdate:
          'Den här uppdateringen installerar kärna {{version}}. Enheten startar om och återgår själv till den nuvarande kärnan om den nya inte startar.',
        rolledBack:
          'Kärnan {{version}} startade inte och enheten återgick till den föregående kärnan.',
        available: 'En uppdatering finns tillgänglig. Vill du uppdatera nu?',
        updating: 'Uppdatering påbörjad. Vänta...',
        confirm: 'Bekräfta',
        cancel: 'Avbryt',
        preview: 'Förhandsvisning av uppdateringar',
        previewDesc: 'Få tidig tillgång till nya funktioner och förbättringar',
        previewTip:
          'Observera att förhandsversioner kan innehålla buggar eller ofullständig funktionalitet!',
        customServer: {
          title: 'Anpassad uppdateringsserver',
          desc: 'Sök efter och hämta onlineuppdateringar från en angiven server',
          invalidUrl:
            'Ange en giltig HTTP- eller HTTPS-serverkatalog utan frågesträng, fragment eller latest.json.',
          loadFailed: 'Det gick inte att läsa in uppdateringsserverns konfiguration.',
          saveFailed: 'Det gick inte att spara uppdateringsserverns konfiguration.',
          saved: 'Uppdateringsserverns konfiguration har sparats.',
          save: 'Spara',
          confirmTitle: 'Vill du använda en anpassad uppdateringsserver?',
          confirmDesc:
            'SHA-512 kontrollerar endast att paketet överensstämmer med manifestet från den här servern. Det bevisar inte att paketet är en officiell NanoKVM-utgåva. En felaktig eller skadlig server kan göra enheten obrukbar, orsaka dataförlust eller äventyra systemets säkerhet.',
          confirm: 'Använd ändå',
          previewDisabled:
            'Förhandsuppdateringar är inte tillgängliga när en anpassad uppdateringsserver är aktiverad.'
        },
        offline: {
          kernelNotice:
            'Det här paketet innehåller en kärna. Den skrivs till reservplatsen och enheten startar om för att prova den; kommer den inte tillbaka återgår enheten själv till den nuvarande kärnan.',
          kernelConfirm: 'Installera kärna',
          kernelCancel: 'Avbryt',
          title: 'Offlineuppdateringar',
          desc: 'Uppdatera genom lokalt installationspaket',
          upload: 'Ladda upp',
          checksumPlaceholder: 'SHA-256-kontrollsumma (valfri)',
          invalidChecksum: 'SHA-256-kontrollsumman måste innehålla 64 hexadecimala tecken.',
          checksumMismatch: 'SHA-256-verifieringen misslyckades. Paketet kan vara skadat.',
          invalidName: 'Ogiltigt filnamnsformat. Ladda ner från GitHub-versioner.',
          updateFailed: 'Uppdatering misslyckades. Försök igen.'
        }
      },
      account: {
        title: 'Konto',
        webAccount: 'Webbkonto-namn',
        role: 'Roll',
        roles: {
          admin: 'Administratör',
          user: 'Användare'
        },
        password: 'Lösenord',
        updateBtn: 'Byt',
        logoutBtn: 'Logga ut',
        logoutDesc: 'Är du säker på att du vill logga ut?',
        okBtn: 'Ja',
        cancelBtn: 'Nej',
        users: {
          title: 'Användare',
          create: 'Skapa användare',
          enabled: 'Aktiverad',
          disabled: 'Inaktiverad',
          deviceOwner: 'Enhetens ägare',
          resetPassword: 'Återställ lösenord',
          delete: 'Ta bort',
          deleteConfirm: 'Ta bort den här användaren och återkalla alla dennes sessioner?',
          created: 'Användare skapad',
          deleted: 'Användare borttagen',
          passwordUpdated: 'Lösenord uppdaterat',
          loadFailed: 'Kunde inte läsa in användare',
          saveFailed: 'Kunde inte spara användaren',
          deleteFailed: 'Kunde inte ta bort användaren'
        }
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistent',
      empty: 'Öppna panelen och starta en uppgift för att börja.',
      inputPlaceholder: 'Beskriv vad du vill att PicoClaw ska göra',
      newConversation: 'Ny konversation',
      processing: 'Bearbetar...',
      agent: {
        defaultTitle: 'Allmän assistent',
        defaultDescription: 'Allmän hjälp för chatt, sökning och arbetsyta.',
        kvmTitle: 'Fjärrstyrning',
        kvmDescription: 'Manövrera fjärrvärden genom NanoKVM.',
        switched: 'Agentroll bytte',
        switchFailed: 'Det gick inte att byta agentroll'
      },
      send: 'Skicka',
      cancel: 'Avbryt',
      status: {
        connecting: 'Ansluter till gateway...',
        connected: 'PicoClaw-session ansluten',
        disconnected: 'PicoClaw-session frånkopplad',
        stopped: 'Stoppbegäran har skickats',
        runtimeStarted: 'PicoClaw runtime startad',
        runtimeStartFailed: 'Det gick inte att starta PicoClaw runtime',
        runtimeStopped: 'PicoClaw runtime stoppad',
        runtimeStopFailed: 'Det gick inte att stoppa PicoClaw runtime',
        controlSwitchedToMCP: 'Styrningen har växlats till den externa MCP-tjänsten'
      },
      connection: {
        runtime: {
          checking: 'Kontrollerar',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime klar',
          stopped: 'Runtime stoppad',
          blockedByMCP: 'Extern MCP-styrning är aktiv',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime inte tillgänglig',
          configError: 'Konfigurationsfel'
        },
        transport: {
          connecting: 'Ansluter',
          connected: 'Ansluten',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Inaktiv',
          busy: 'Upptagen'
        }
      },
      message: {
        toolAction: 'Åtgärd',
        observation: 'Observation',
        screenshot: 'Skärmdump'
      },
      overlay: {
        locked: 'PicoClaw styr enheten. Manuell inmatning är pausad.'
      },
      control: {
        picoclaw: 'Enhetsstyrning: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Enhetsstyrning: extern MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Enhetsstyrning: av',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Ge styrning',
        release: 'Släpp',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw-styrning beviljad',
        released: 'PicoClaw-styrning släppt',
        grantFailed: 'Det gick inte att ge PicoClaw styrning',
        releaseFailed: 'Det gick inte att släppa PicoClaw styrning',
        grantConfirmTitle: 'Växla enhetsstyrning till PicoClaw?',
        grantConfirmDesc: 'Externa MCP-enhetsskrivningar kommer att avbrytas.'
      },
      install: {
        install: 'Installera PicoClaw',
        installing: 'Installerar PicoClaw',
        success: 'PicoClaw har installerats framgångsrikt',
        failed: 'Det gick inte att installera PicoClaw',
        uninstalling: 'Avinstallerar runtime...',
        uninstalled: 'Runtime avinstallerades framgångsrikt.',
        uninstallFailed: 'Avinstallationen misslyckades.',
        requiredTitle: 'PicoClaw är inte installerad',
        requiredDescription: 'Installera PicoClaw innan du startar PicoClaw runtime.',
        progressDescription: 'PicoClaw laddas ner och installeras.',
        stages: {
          preparing: 'Förbereder',
          downloading: 'Laddar ner',
          extracting: 'Packar upp',
          verifying: 'Verifierar',
          installing: 'Installerar',
          installed: 'Installerad',
          install_timeout: 'Timeout',
          install_failed: 'Misslyckades'
        }
      },
      model: {
        requiredTitle: 'Modellkonfiguration krävs',
        requiredDescription: 'Konfigurera PicoClaw-modellen innan du använder PicoClaw-chatten.',
        docsTitle: 'Konfigurationsguide',
        docsDesc: 'Modeller och protokoll som stöds',
        menuLabel: 'Konfigurera modell',
        modelIdentifier: 'Modellidentifierare',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API-nyckel',
        apiKeyPlaceholder: 'Ange modellens API-nyckel',
        save: 'Spara',
        saving: 'Sparar',
        saved: 'Modellkonfiguration sparad',
        saveFailed: 'Det gick inte att spara modellkonfigurationen',
        invalid: 'Modellidentifierare, API Base URL och API-nyckel krävs'
      },
      uninstall: {
        menuLabel: 'Avinstallera',
        confirmTitle: 'Avinstallera PicoClaw',
        confirmContent:
          'Är du säker på att du vill avinstallera PicoClaw? Detta kommer att radera den körbara filen och alla konfigurationsfiler.',
        confirmOk: 'Avinstallera',
        confirmCancel: 'Avbryt'
      },
      history: {
        title: 'Historik',
        loading: 'Laddar sessioner...',
        emptyTitle: 'Ingen historik än',
        emptyDescription: 'Tidigare PicoClaw-sessioner kommer att visas här.',
        loadFailed: 'Det gick inte att ladda sessionshistoriken',
        deleteFailed: 'Det gick inte att ta bort sessionen',
        deleteConfirmTitle: 'Ta bort session',
        deleteConfirmContent: 'Är du säker på att du vill ta bort "{{title}}"?',
        deleteConfirmOk: 'Ta bort',
        deleteConfirmCancel: 'Avbryt',
        messageCount_one: '{{count}} meddelande',
        messageCount_other: '{{count}} meddelanden',
        messageCount: '{{count}} meddelanden'
      },
      config: {
        startRuntime: 'Starta PicoClaw',
        stopRuntime: 'Stoppa PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Växla styrningen till PicoClaw?',
        enableConfirmDesc: 'När PicoClaw startas inaktiveras den externa MCP-tjänsten.',
        enableConfirmOk: 'Starta PicoClaw',
        enableConfirmCancel: 'Avbryt',
        title: 'Starta PicoClaw',
        description: 'Starta runtime för att börja använda PicoClaw-assistenten.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Vi stötte på ett problem',
      refresh: 'Uppdatera'
    },
    fullscreen: {
      toggle: 'Växla fullskärm'
    },
    menu: {
      collapse: 'Fäll ihop menyn',
      expand: 'Expandera menyn'
    }
  }
};

export default se;
