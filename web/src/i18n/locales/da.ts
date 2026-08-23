const da = {
  translation: {
    head: {
      desktop: 'Fjernskrivebord',
      login: 'Log ind',
      changePassword: 'Skift adgangskode',
      terminal: 'Terminal',
      wifi: 'Wi-Fi'
    },
    auth: {
      login: 'Log ind',
      placeholderUsername: 'Indtast brugernavn',
      placeholderPassword: 'indtast adgangskode',
      placeholderPassword2: 'indtast adgangskode igen',
      noEmptyUsername: 'brugernavn kan ikke være tom',
      noEmptyPassword: 'adgangskode kan ikke være tom',
      noAccount:
        'Kunne ikke hente brugeroplysninger. Prøv at opdater siden eller nulstil adgangskoden',
      invalidUser: 'ugyldigt brugernavn eller adgangskode',
      locked: 'For mange logins, prøv venligst igen senere',
      globalLocked: 'System under beskyttelse, prøv venligst igen senere',
      error: 'uventet fejl',
      changePassword: 'Skift adgangskode',
      changePasswordDesc: 'For sikkerheden af din enhed, bedes du ændre web-login adgangskoden.',
      differentPassword: 'Adgangskoder er ikke ens',
      illegalUsername: 'brugernavn indeholder ugyldige tegn',
      illegalPassword: 'adgangskode indeholder ugyldige tegn',
      forgetPassword: 'Glem adgangskode',
      ok: 'OK',
      cancel: 'Annuller',
      loginButtonText: 'Log ind',
      tips: {
        reset1:
          'To reset the passwords, pressing and holding the BOOT button on the NanoKVM for 10 seconds.',
        reset2: 'Se detaljerede trin i dette dokument:',
        reset3: 'Standard webkonto:',
        reset4: 'Standard SSH-konto:',
        change1: 'Bemærk, at denne handling ændrer følgende adgangskoder:',
        change2: 'Adgangskode til weblogin',
        change3: 'Systemets root-adgangskode (SSH-loginadgangskode)',
        change4: 'For at nulstille adgangskoderne skal du holde BOOT-knappen på NanoKVM nede.'
      }
    },
    wifi: {
      title: 'Wi-Fi',
      description: 'Konfigurer Wi-Fi for NanoKVM',
      success: 'Please check the network status of NanoKVM and visit the new IP address.',
      failed: 'Handlingen mislykkedes, prøv igen.',
      invalidMode:
        'Den aktuelle tilstand understøtter ikke netværksopsætning. Gå til din enhed og aktiver Wi-Fi konfigurationstilstand.',
      confirmBtn: 'Ok',
      finishBtn: 'Færdig',
      ap: {
        authTitle: 'Godkendelse påkrævet',
        authDescription: 'Indtast venligst AP adgangskoden for at fortsætte',
        authFailed: 'Ugyldig AP adgangskode',
        passPlaceholder: 'AP adgangskode',
        verifyBtn: 'Bekræft'
      }
    },
    screen: {
      scale: 'Skala',
      title: 'Skærm',
      video: 'Videotilstand',
      videoDirectTips: 'Aktiver HTTPS i "Indstillinger > Enhed" for at bruge denne tilstand',
      resolution: 'Opløsning',
      auto: 'Automatisk',
      autoTips:
        'Screen-tearing eller mouse-offset kan opstå ved enkelte opløsninger. Hvis du oplever dette, kan du prøve at justere fjerncomputerens skærmopløsning eller deaktivere automatisk tilstand.',
      fps: 'FPS',
      customizeFps: 'Tilpas',
      quality: 'Kvalitet',
      qualityLossless: 'Tabsfri',
      qualityHigh: 'Høj',
      qualityMedium: 'Mellem',
      qualityLow: 'Lav',
      frameDetect: 'Beregn frames',
      frameDetectTip:
        'Beregner forskellen mellem hver frame. Stopper med at sende et video stream hvis der ikke registreres ændringer på fjerncomputerens skærm.',
      resetHdmi: 'Nulstil HDMI',
      mixedH264: {
        title: 'H.264-streamkonflikt',
        description:
          'H.264 Direct og H.264 WebRTC bruges samtidigt. Dette kan forårsage skærmrivning eller beskadiget video. Brug kun én H.264-tilstand.'
      },
      webrtcConnectionFailed: {
        title: 'WebRTC-forbindelse mislykkedes',
        description: 'Kontrollér netværksforbindelsen, eller skift videotilstand.'
      },
      captureStatus: {
        hdmiError: 'Fejl i HDMI-billedet',
        unsupportedResolution: 'Den aktuelle opløsning understøttes ikke',
        retrieving: 'Henter skærmbillede...',
        changingResolution: 'Skifter opløsning...',
        updateFailed: 'Skærmbilledet kan ikke opdateres lige nu',
        videoError: 'Fejl i videovisning',
        noHdmi: 'Intet HDMI-signal registreret',
        unavailable: 'Skærmbilledet kan ikke vises lige nu'
      }
    },
    keyboard: {
      title: 'Tastatur',
      paste: 'Indsæt',
      tips: 'Kun standard bogstaver og symboler er understøttet',
      placeholder: 'Indtast tekst',
      submit: 'Send',
      virtual: 'Tastatur',
      readClipboard: 'Læs fra udklipsholder',
      clipboardPermissionDenied:
        'Udklipsholdertilladelse nægtet. Tillad venligst udklipsholderadgang i din browser.',
      clipboardReadError: 'Kunne ikke læse udklipsholderen',
      dropdownEnglish: 'Engelsk',
      dropdownGerman: 'tysk',
      dropdownFrench: 'Fransk',
      dropdownRussian: 'russisk',
      shortcut: {
        title: 'Genveje',
        custom: 'Brugerdefineret',
        capture: 'Klik her for at fange genvej',
        clear: 'Ryd',
        save: 'Gem',
        captureTips:
          'Optagelse af systemtaster (såsom Windows-tasten) kræver fuldskærmstilladelse.',
        enterFullScreen: 'Skift fuldskærmstilstand.'
      },
      leaderKey: {
        title: 'Leader-tast',
        desc: 'Omgå browserbegrænsninger og send systemgenveje direkte til fjernværten.',
        howToUse: 'Sådan bruges',
        simultaneous: {
          title: 'Samtidig tilstand',
          desc1: 'Hold Leader-tasten nede, og tryk derefter på genvejen.',
          desc2: 'Intuitivt, men kan være i konflikt med systemgenveje.'
        },
        sequential: {
          title: 'Sekventiel tilstand',
          desc1:
            'Tryk på Leader-tasten → tryk på genvejen i rækkefølge → tryk på Leader-tasten igen.',
          desc2: 'Kræver flere trin, men undgår fuldstændig systemkonflikter.'
        },
        enable: 'Aktiver Leader-tast',
        tip: 'Når denne tast tildeles som Leader-tast, fungerer den kun som genvejsudløser og mister sin standardfunktion.',
        placeholder: 'Tryk på Leader-tasten',
        shiftRight: 'Højre Shift',
        ctrlRight: 'Højre Ctrl',
        metaRight: 'Højre Win',
        submit: 'Send',
        recorder: {
          rec: 'REC',
          activate: 'Aktiver taster',
          input: 'Tryk på genvejen...'
        }
      }
    },
    mouse: {
      title: 'Mus',
      cursor: 'Markørstil',
      default: 'Standard-markør',
      pointer: 'Peger-markør',
      cell: 'Celle-markør',
      text: 'Tekst-markør',
      grab: 'Grib-markør',
      hide: 'Skjul mus',
      mode: 'Tilstand for mus',
      absolute: 'Absolut tilstand',
      relative: 'Relativ tilstand',
      direction: 'Rullehjulsretning',
      scrollUp: 'Rul op',
      scrollDown: 'Rul ned',
      speed: 'Rullehjulshastighed',
      fast: 'Hurtigt',
      slow: 'Langsomt',
      requestPointer: 'Bruger relativ-tilstand. Klik på skrivebordet for at få musemarkør.',
      resetHid: 'Nulstil HID',
      hidOnly: {
        title: 'Kun HID-tilstand',
        desc: 'Hvis din mus og tastatur holder op med at reagere, og nulstilling af HID ikke hjælper, kan det være et kompatibilitetsproblem mellem NanoKVM og enheden. Prøv at aktivere HID-Only-tilstand for bedre kompatibilitet.',
        tip1: 'Aktivering af HID-Only-tilstand vil afmontere den virtuelle U-disk og det virtuelle netværk',
        tip2: 'I HID-Only-tilstand er billedmontering deaktiveret',
        tip3: 'NanoKVM genstarter automatisk efter at have skiftet tilstand',
        enable: 'Aktiver HID-kun tilstand',
        disable: 'Deaktiver HID-kun tilstand'
      }
    },
    image: {
      title: 'Diskbilleder',
      loading: 'Kontrollerer...',
      empty: 'Ingen fundet',
      mountMode: 'Monteringstilstand',
      mountFailed: 'Montering af diskbillede mislykkedes',
      mountDesc:
        'På nogle systemer kan det være nødvendigt at skubbe den virtuelle disk ud på fjerncomputeren før du kan montere diskbilledet.',
      unmountFailed: 'Afmontering mislykkedes',
      unmountDesc:
        'På nogle systemer skal du manuelt skubbe ud fra fjernværten, før du afmonterer billedet.',
      refresh: 'Opdater billedlisten',
      attention: 'Opmærksomhed påkrævet',
      deleteConfirm: 'Er du sikker på, at du vil slette dette billede?',
      okBtn: 'Ja',
      cancelBtn: 'Annuller',
      tips: {
        title: 'Sådan uploader du',
        usb1: 'Forbind din NanoKVM til din computer via USB.',
        usb2: 'Sørg for, at den virtuelle disk er monteret (Indstillinger -> Virtuel disk).',
        usb3: 'Åben den virtuelle disk på din computer og kopier diskbilledet til roden af den virtuelle disk.',
        scp1: 'Kontroller at din NanoKVM og din computer er på samme lokale netværk.',
        scp2: 'Åben en terminal på din computer og brug SCP-kommandoen for at uploade diskbilledet til /data mappen på din NanoKVM.',
        scp3: 'Eksempel: scp sti-til-dit-diskbillede root@din-nanokvm-ip:/data',
        tfCard: 'microSD-kort',
        tf1: 'Denne metode er understøttet af Linux systemer',
        tf2: 'Tag microSD-kortet ud af din NanoKVM (for den fulde version af NanoKVM skal du åbne enheden for at kunne tage microSD-kortet ud).',
        tf3: 'Indsæt microSD-kortet i en kortlæser og tilslut den til en computer.',
        tf4: 'Kopier diskbilledet til /data mappen på microSD-kortet.',
        tf5: 'Skub microSD-kortet ud og indsæt microSD-kortet i din NanoKVM.'
      }
    },
    script: {
      title: 'Script',
      upload: 'Upload',
      run: 'Kør',
      runBackground: 'Kør i baggrunden',
      runFailed: 'Kørsel mislykkedes',
      attention: 'Opmærksomhed påkrævet',
      delDesc: 'Er du sikker på at du vil slette denne fil?',
      confirm: 'Ja',
      cancel: 'Annuller',
      delete: 'Slet',
      close: 'Luk'
    },
    terminal: {
      title: 'Terminal',
      nanokvm: 'Terminal til NanoKVM',
      serial: 'Terminal til seriel port',
      serialPort: 'Serial port',
      serialPortPlaceholder: 'Angiv seriel port',
      baudrate: 'Baud-hastighed',
      parity: 'Paritet',
      parityNone: 'Ingen',
      parityEven: 'Lige',
      parityOdd: 'Ulige',
      flowControl: 'Flowkontrol',
      flowControlNone: 'Ingen',
      flowControlSoft: 'Software',
      flowControlHard: 'Hardware',
      dataBits: 'Databits',
      stopBits: 'Stopbit',
      confirm: 'OK'
    },
    wol: {
      title: 'Wake-on-LAN',
      sending: 'Sender Wake-on-LAN magic packet',
      sent: 'Wake-on-LAN magic packet sendt',
      input: 'Angiv MAC-adresse',
      ok: 'OK'
    },
    download: {
      title: 'Billedhenter',
      input: 'Indtast venligst et fjernbillede URL',
      ok: 'OK',
      disabled: '/data partitionen er RO, så vi kan ikke downloade billedet',
      uploadbox: 'Slip filen her, eller klik for at vælge',
      inputfile: 'Indtast venligst billedfilen',
      NoISO: 'Ingen ISO',
      sha256: 'SHA-256 (valgfri)',
      sha256Placeholder: 'Indtast en SHA-256-kontrolsum på 64 tegn',
      invalidSHA256: 'SHA-256 skal være en hexadecimal streng på 64 tegn',
      failed: 'Download mislykkedes',
      success: 'Download gennemført',
      checksumFailed: 'Download mislykkedes: SHA-256-verifikation mislykkedes',
      cancel: 'Annuller',
      cancelFailed: 'Kunne ikke annullere download'
    },
    power: {
      title: 'Tænd/sluk-knap',
      showConfirm: 'Bekræftelse',
      showConfirmTip: 'Strømdrift kræver en ekstra bekræftelse',
      reset: 'Nulstillingsknap',
      power: 'Tænd/sluk-knap',
      powerShort: 'Tænd/sluk-knap (kort tryk)',
      powerLong: 'Tænd/sluk-knap (langt tryk)',
      resetConfirm: 'Fortsæt med nulstilling?',
      powerConfirm: 'Fortsæt strømdrift?',
      okBtn: 'Ja',
      cancelBtn: 'Annuller'
    },
    devices: {
      takeover: 'Overtag',
      refused: 'Bruges af {{owner}} fra {{source}}',
      share: {
        usbDevice: 'Del USB'
      },
      permission: {
        denied: 'Blokeret i browserens webstedsindstillinger',
        prompt: 'Browseren vil bede om adgang'
      },
      mic: {
        mute: 'Slå lyden fra',
        unmute: 'Slå lyden til'
      },
      revoked: {
        released: 'Delingen blev stoppet',
        lease_expired: 'Lejemålet udløb, før denne browser kom tilbage',
        admin_disconnect: 'En administrator afbrød alle kilder',
        slot_removed: 'Pladsen blev fjernet',
        slot_changed: 'Pladsen blev konfigureret om',
        taken_over: 'En administrator overtog denne plads'
      },
      usb: {
        surrendered: 'USB-passthrough holder tastaturet og musen',
        surrenderedDesc:
          'Fjernværten ser den importerede enhed i stedet for NanoKVM’s tastatur, mus og virtuelle medier. De kommer tilbage, når sessionen stopper.',
        unsupported: 'WebUSB kræver en Chromium-browser over HTTPS',
        session: 'Videresender {{device}} ({{mode}})',
        idle: 'Ingen passthrough-session',
        mode: {
          hybrid: 'hybrid',
          exact: 'eksakt'
        }
      }
    },
    settings: {
      title: 'Indstillinger',
      display: {
        title: 'Skærm',
        loading: 'Indlæser...',
        active: 'Aktiv EDID',
        activeUnknown:
          'NanoKVM har ikke skrevet en EDID, siden den startede, så det er ukendt, hvilken skærm værten ser.',
        appliedAt: 'Anvendt {{time}}',
        download: 'Hent',
        downloadBackup: 'Hent den forrige',
        preset: 'Skærmforudindstilling',
        presetPlaceholder: 'Vælg en skærm',
        upload: 'Upload',
        selected: 'Valgt EDID',
        errors: 'Fejl',
        warnings: 'Advarsler',
        info: 'Information',
        unknownMonitor: 'Ukendt skærm',
        edidVersion: 'EDID {{version}}',
        audioYes: 'Lyd',
        audioNo: 'Ingen lyd',
        extensionBlocks: 'Udvidelsesblokke: {{blocks}}',
        apply: 'Anvend',
        applyTitle: 'Anvend denne EDID?',
        before: 'Nuværende',
        after: 'Ny',
        hdmiNotice:
          'Videooptagelsen stopper, mens EDID skrives, og starter af sig selv igen bagefter.',
        powerCycleNotice:
          'Enheden skal fysisk tages ud af stikkontakten og sættes i igen, før den nye EDID træder i kraft.',
        powerCycleUnverified:
          'Skrivningen blev ikke verificeret, så videochippen beholder det, den nu indeholder, indtil denne enhed fysisk kobles fra strømmen og tilsluttes igen.',
        applied: 'EDID anvendt og verificeret.',
        applyFailed: 'Det lykkedes ikke at anvende EDID.',
        busy: 'Videochippen var optaget. Prøv igen.',
        unsupported: 'Denne enhed understøtter ikke ændring af EDID.',
        toolMissing: 'EDID-værktøjet mangler i denne firmware.',
        noAudio: 'Denne EDID annoncerer ingen lyd, så værten holder måske op med at sende lyd.',
        oldVersion: 'Denne EDID bruger en ældre version end 1.4.',
        interlaced: 'Den foretrukne opløsning er interlaced.',
        tooLarge:
          'Den foretrukne opløsning er større end 1920x1080 ved 60 Hz, hvilket er mere, end NanoKVM kan optage.',
        recovery: 'Gendannelse',
        recoveryNeeded:
          'Den seneste skrivning blev ikke verificeret, så EDID-området i videochippen er i en ukendt tilstand. Gendan fabriks-EDID for at gøre tilstanden kendt igen.',
        recoveryDesc:
          'Skriv et kendt EDID tilbage til videochippen, hvis et anvendt EDID efterlod værten uden billede.',
        restoreFactory: 'Gendan fabriks-EDID',
        restoreBackup: 'Gendan forrige EDID',
        restoreTitle: 'Gendan dette EDID?',
        restoreFactoryTarget: 'Det fabriks-EDID, som NanoKVM blev leveret med.',
        restoreBackupTarget: 'Den nyeste sikkerhedskopi, anvendt {{time}}.',
        restoreNotice:
          'En gendannelse skrives på samme måde som en anvendelse og har de samme konsekvenser.',
        restored: 'EDID gendannet og verificeret.',
        restoreFailed: 'Gendannelse af EDID mislykkedes.',
        mismatchTitle: 'Skrevet og læst tilbage',
        mismatchWritten: 'Skrevet',
        mismatchRead: 'Læst tilbage',
        restoreOkBtn: 'Gendan',
        hardware: 'Registreret hardware: {{hardware}}',
        hardwareUnknown: 'Ukendt',
        confirmWord: 'ANVEND',
        confirmPrompt: 'Skriv {{word}} for at aktivere knappen Anvend.',
        okBtn: 'Anvend',
        cancelBtn: 'Annuller'
      },
      presentation: {
        noProfile: 'Ingen profil anvendt',
        linked: 'Tilknyttede funktioner',
        hostState: 'Værtens USB',
        hostUnbound: 'Controlleren er ikke bundet',
        hdmiState: 'HDMI-indgang',
        hdmiSignal: 'Signal til stede',
        hdmiUnreported: 'Endnu ingen melding om optagelse',
        endpoints: 'Endpoints',
        fifos: 'FIFO-pladser',
        pending: 'Ventende ændringer',
        pendingEdits: 'Ikke-gemte identitetsændringer',
        pendingProfile: '{{profile}} er valgt, men ikke anvendt',
        pendingNone: 'Ingen',
        lastApply: 'Seneste anvendelse',
        applyFailed: 'Mislykkedes på {{profile}} den {{time}}',
        applyClean: 'Ingen fejl registreret',
        lastKnownGood: 'Senest kendte fungerende',
        rollbackTarget: 'Mål for tilbagerulning',
        rollbackNone: 'Ingen',
        powerCyclePending:
          'Controlleren blev taget fra værten. Sluk og tænd den tilsluttede computer for at få enheden tilbage.',
        rollback: 'Rul tilbage',
        rollbackTitle: 'Rul tilbage til {{profile}}?',
        rollbackDesc: 'Gadgetten opregnes på ny; USB-funktioner falder kortvarigt ud.',
        descriptors: 'deskriptorer',
        applyLinks: 'Tilknytter: {{functions}}',
        applyRemoves: 'Fjerner: {{functions}}',
        applyNoHid:
          'Der er ingen HID-funktion tilbage efter denne anvendelse. Tastatur og mus holder op med at virke.',
        applyRollback: 'En mislykket anvendelse vender tilbage til {{profile}}.',
        recoveryPowerCycle:
          'Ingen HID overlever denne anvendelse, så en vært, der holder op med at svare, kan kun reddes ved at slukke og tænde den.',
        recoveryReboot:
          'En grænseflade forsvinder fra den sammensatte enhed; værten kan have brug for en genstart for at binde resten igen.',
        recoveryHdmiReset: 'En videofunktion bygges op igen, så optagekæden bag den nulstilles.',
        recoveryReconnect: 'Værten opregner enheden på ny; USB-funktioner falder kortvarigt ud.'
      },
      passthrough: {
        title: 'USB-passthrough',
        loading: 'Indlæser...',
        hidWarning: 'At starte passthrough afgiver tastaturet, musen og virtuelle medier',
        hidWarningDesc:
          'NanoKVM har kun én USB-enhedscontroller, og proxyen skal bruge den hele. Mens en session kører, ser den fjerne vært derfor den videregivne enhed i stedet for NanoKVM’s tastatur, mus og virtuelle medier. De kommer af sig selv tilbage i samme øjeblik sessionen stoppes. Denne webgrænseflade er ikke berørt, så du kan altid stoppe en session fra denne side.',
        isoWarning: 'Webkameraer, mikrofoner og andre isokrone enheder kan ikke videregives',
        isoWarningDesc:
          'Denne hardware kan kun bære control-, bulk- og interrupt-overførsler. Lyd- og videoenheder virker ikke, uanset hvordan de bindes.',
        session: 'Session',
        activeDesc: 'En enhed er importeret, og proxyen holder USB-controlleren.',
        inactiveDesc: 'Ingen session kører. Tastatur, mus og virtuelle medier fungerer normalt.',
        device: 'Enhed',
        busId: 'Bus-id',
        speed: 'Hastighed',
        exporter: 'Eksportør',
        local: 'Importeret som',
        localValue: 'Bus {{bus}}, adresse {{address}}',
        udc: 'USB-controller',
        pid: 'Proxy-PID',
        startedAt: 'Startet',
        isoDevice:
          'Denne enhed melder en lyd- eller videoklasse, som kræver isokrone overførsler. Den kommer ikke til at virke.',
        exporterLabel: 'Eksportørens adresse',
        exporterHint:
          'Værten og porten, som NanoKVM ringer op til. Gennem tunnelen nedenfor er det {{exporter}}.',
        busIdLabel: 'Bus-id på din egen maskine',
        busIdHint: 'Det busid, som usbip list -l viser for enheden, for eksempel {{example}}.',
        start: 'Start passthrough',
        stop: 'Stop passthrough',
        startTitle: 'Vil du starte USB-passthrough?',
        startDevice: 'NanoKVM importerer {{busId}} fra {{exporter}}.',
        startHid:
          'USB-tastaturet, musen og virtuelle medier holder op med at virke, så længe sessionen kører, og starter af sig selv igen, når du stopper den.',
        startIso:
          'Webkameraer, mikrofoner og andre isokrone enheder virker ikke på denne hardware.',
        startWeb:
          'Denne webgrænseflade bliver ved med at virke, så du kan stoppe sessionen fra denne side når som helst.',
        okBtn: 'Start',
        cancelBtn: 'Annuller',
        instructions: 'På din egen maskine',
        instructionsDesc:
          'Der er bevidst ingen klientagent at installere. Kør disse almindelige usbip-kommandoer på den maskine, enheden sidder i.',
        copyFailed: 'Kopiering mislykkedes. Kopier kommandoen manuelt.',
        directNote:
          'Uden tunnel skal usbipd kunne nås på dit netværk, og eksportøradressen ovenfor skal pege på den. usbip sender enheden ukrypteret, så tunnelen er at foretrække.',
        steps: {
          modprobe: {
            title: 'Indlæs driveren i eksportsiden',
            desc: 'usbip-host er det, der lader din kerne aflevere en lokal enhed. Den indlæses ikke som standard.'
          },
          list: {
            title: 'Find enheden',
            desc: 'Viser hver lokal enhed med dens busid og dens producent:produkt-par. Notér busid for den, du vil bruge.'
          },
          bind: {
            title: 'Bind den til usbip',
            desc: 'Tager enheden fra dens normale driver, så den holder op med at virke på denne maskine, indtil du løsner bindingen.'
          },
          serve: {
            title: 'Stil den til rådighed',
            desc: 'usbipd bliver i forgrunden og venter på, at NanoKVM importerer enheden.',
            notice:
              'Den almindelige usbipd har ingen indstilling for lytteadresse og lytter på alle grænseflader. Hold port {{port}} lukket i din firewall, og lad kun tunnelen nedenfor nå den.'
          },
          tunnel: {
            title: 'Peg den mod NanoKVM',
            desc: 'En omvendt SSH-tunnel: port {{port}} på NanoKVM’s egen loopback bliver til usbipd på denne maskine. Lad den køre hele sessionen.'
          },
          exporter: {
            title: 'Brug dette som eksportør',
            desc: 'Skriv dette i eksportørfeltet ovenfor, indtast bus-id og start sessionen.'
          },
          unbind: {
            title: 'Giv enheden tilbage',
            desc: 'Når sessionen er stoppet, giver dette enheden tilbage til dens normale driver på denne maskine.'
          }
        }
      },
      mcp: {
        title: 'MCP-tjeneste',
        service: 'MCP-fjernbetjening',
        serviceDesc: 'Tillad betroede MCP-klienter at styre tastatur og mus og tage skærmbilleder',
        securityWarning:
          'Alle med denne API-nøgle kan styre fjernværten og se dens skærm. Brug HTTPS, og aktivér kun tjenesten på netværk, du har tillid til.',
        endpoint: 'Slutpunkt',
        apiKey: 'API-nøgle',
        regenerateConfirmTitle: 'Generér MCP API-nøglen igen?',
        regenerateConfirmDesc: 'Den nuværende nøgle holder straks op med at virke.',
        enableConfirmTitle: 'Aktivér ekstern MCP-styring?',
        enableConfirmDesc:
          'Aktivering af MCP stopper PicoClaw og lukker alle aktive PicoClaw-sessioner.',
        failed: 'MCP-handlingen mislykkedes',
        copyFailed: 'Kopiering mislykkedes. Kopiér manuelt.',
        okBtn: 'Bekræft',
        cancelBtn: 'Annuller'
      },
      about: {
        title: 'Om NanoKVM',
        information: 'Information',
        ip: 'IP',
        mdns: 'mDNS',
        application: 'Program version',
        applicationTip: 'Version af NanoKVM-webapplikationen',
        image: 'Firmware version',
        imageTip: 'Version af NanoKVM-systemimaget',
        deviceKey: 'Enhedsnøgle',
        community: 'Fællesskab',
        hostname: 'Værtsnavn',
        hostnameUpdated: 'Værtsnavn opdateret. Genstart for at anvende.',
        ipType: {
          Wired: 'Kablet',
          Wireless: 'Trådløs',
          Other: 'Andet'
        }
      },
      appearance: {
        title: 'Udseende',
        display: 'Visning',
        language: 'Sprog',
        languageDesc: 'Vælg sproget til grænsefladen',
        webTitle: 'Webtitel',
        webTitleDesc: 'Tilpas websidens titel',
        menuBar: {
          title: 'Menulinje',
          mode: 'Visningstilstand',
          modeDesc: 'Vis menulinje på skærmen',
          modeOff: 'Fra',
          modeAuto: 'Skjul automatisk',
          modeAlways: 'Altid synlig',
          keyboardLedStatus: 'Tastaturlåseindikatorer',
          keyboardLedStatusDesc:
            'Vis Num Lock-, Caps Lock- og Scroll Lock-status for fjerncomputeren',
          icons: 'Undermenuikoner',
          iconsDesc: 'Vis undermenuikoner i menulinjen'
        }
      },
      keyboardLedStatus: {
        groupLabel: 'Status for låse på fjernkeyboard',
        indicatorLabel: '{{label}}: {{state}}',
        numLock: 'Num Lock',
        numLockShort: 'Num',
        capsLock: 'Caps Lock',
        capsLockShort: 'Caps',
        scrollLock: 'Scroll Lock',
        scrollLockShort: 'Scr',
        on: 'Til',
        off: 'Fra',
        unknown: 'Ukendt'
      },
      device: {
        title: 'Enhed',
        oled: {
          title: 'OLED',
          description: 'OLED screen automatically sleep',
          0: 'Aldrig',
          15: '15 sek.',
          30: '30 sek.',
          60: '1 min',
          180: '3 min',
          300: '5 min',
          600: '10 min',
          1800: '30 min',
          3600: '1 time'
        },
        ssh: {
          description: 'Aktiver SSH fjernadgang',
          tip: 'Indstil en stærk adgangskode før aktivering (Konto - Skift adgangskode)'
        },
        advanced: 'Avancerede indstillinger',
        swap: {
          title: 'Byt',
          disable: 'Deaktiver',
          description: 'Indstil swap-filstørrelsen',
          tip: 'Aktivering af denne funktion kan forkorte dit SD-korts brugbare levetid!'
        },
        mouseJiggler: {
          title: 'Mus Jiggler',
          description: 'Forhindrer fjernværten i at sove',
          disable: 'Deaktiver',
          absolute: 'Absolut tilstand',
          relative: 'Relativ tilstand'
        },
        mdns: {
          description: 'Aktiver mDNS opdagelsestjeneste',
          tip: 'Slukker den, hvis den ikke er nødvendig'
        },
        hdmi: {
          description: 'Aktiver HDMI/monitor output',
          idleTimeoutTitle: 'Timeout for inaktiv optagelse',
          idleTimeoutDescription: 'Stop HDMI-optagelse efter en periode uden aktive seere på',
          minutes: 'min'
        },
        autostart: {
          title: 'Indstillinger for autostart scripts',
          description: 'Administrer scripts, der kører automatisk ved systemstart',
          new: 'Ny',
          deleteConfirm: 'Er du sikker på at du vil slette denne fil?',
          yes: 'Ja',
          no: 'Annuller',
          scriptName: 'Autostart scriptnavn',
          scriptContent: 'Autostart scriptindhold',
          settings: 'Indstillinger'
        },
        hidOnly: 'HID-Kun tilstand',
        hidOnlyDesc:
          'Stop med at emulere virtuelle enheder, og behold kun grundlæggende HID kontrol',
        disk: 'Virtuel disk',
        diskDesc: 'Mount virtual U-disk on the remote host',
        network: 'Virtuelt netværk',
        networkDesc: 'Monter det virtuelle netværkskort på den eksterne vært',
        networkProtocol: 'Netværksprotokol',
        networkProtocolDesc: 'NCM til moderne værter, RNDIS til ældre Windows',
        reboot: 'Genstart',
        rebootDesc: 'Er du sikker på, at du vil genstarte NanoKVM?',
        okBtn: 'Ja',
        cancelBtn: 'Annuller'
      },
      network: {
        title: 'Netværk',
        wifi: {
          title: 'Wi-Fi',
          description: 'Konfigurer Wi-Fi',
          apMode: 'AP-tilstand er aktiveret, opret forbindelse til Wi-Fi ved at scanne QR-koden',
          connect: 'Tilslut Wi-Fi',
          connectDesc1: 'Indtast netværkets SSID og adgangskode',
          connectDesc2: 'Indtast adgangskoden for at tilslutte dette netværk',
          disconnect: 'Er du sikker på, at du vil afbryde netværket?',
          failed: 'Forbindelsen mislykkedes, prøv igen.',
          ssid: 'Navn',
          password: 'Adgangskode',
          joinBtn: 'Tilslut',
          confirmBtn: 'OK',
          cancelBtn: 'Annuller'
        },
        tls: {
          description: 'Aktiver HTTPS-protokol',
          tip: 'Bemærk: Brug af HTTPS kan øge forsinkelsen, især med MJPEG-videotilstand.'
        },
        bridge: {
          title: 'Netværksbro',
          twoDevices:
            'Din router ser NanoKVM og den styrede computer som to separate enheder, hver med sin egen adresse.',
          loading: 'Indlæser...',
          state: 'Status',
          states: {
            disabled: 'Deaktiveret',
            enabled: 'Aktiveret',
            rolledBack: 'Rullet tilbage',
            failed: 'Mislykkedes',
            pending: 'I gang'
          },
          uplink: 'Uplink',
          ports: 'Porte',
          protocol: 'Enhedens protokol',
          up: 'aktiv',
          down: 'inaktiv',
          noLink: 'intet link',
          enableTitle: 'Aktivér netværksbroen?',
          disableTitle: 'Deaktivér netværksbroen?',
          reconnect:
            'Administrationsforbindelsen afbrydes kortvarigt og genoprettes, mens adressen flyttes.',
          rollback:
            'Hvis verifikationen mislykkes, gendannes den tidligere konfiguration automatisk.',
          enableBtn: 'Aktivér',
          disableBtn: 'Deaktivér',
          cancelBtn: 'Annuller',
          interrupted:
            'Forbindelsen blev afbrudt under anvendelsen. Den aktuelle status kontrolleres igen.',
          pendingNotice:
            'En ændring af broen er stadig i gang eller blev afbrudt, før den blev færdig.',
          revert: 'Gendan tidligere konfiguration',
          rolledBackNotice:
            'Den seneste ændring blev rullet tilbage, og den tidligere konfiguration blev gendannet.',
          verifyFailed: 'Verifikationen mislykkedes: {{gates}}',
          gates: {
            address: 'adresse',
            gateway: 'gateway',
            inbound: 'indgående forbindelse'
          },
          inboundWeak:
            'Kontrollen af indgående trafik lykkedes kun, fordi NanoKVM forbandt til sig selv. Det viser, at webtjenesten lytter og kan nås lokalt, ikke at en forespørgsel fra netværket når frem.',
          noCarrier:
            'Intet link på {{port}}. Broen har ingen vej ud på netværket, før der tilsluttes et kabel.',
          loop: 'Routeren læres også på {{port}}, så den port er en anden vej til det samme netværk. Spanning tree er slået fra, så intet her bryder løkken: afbryd en af de to veje.',
          failedNotice:
            'Den seneste ændring kunne ikke fortrydes. NanoKVM kan muligvis kun nås via Wi-Fi-adgangspunktet eller en seriel konsol.'
        },
        dns: {
          title: 'DNS',
          description: 'Konfigurer DNS-servere til NanoKVM',
          mode: 'Tilstand',
          dhcp: 'DHCP',
          manual: 'Manuel',
          add: 'Tilføj DNS',
          save: 'Gem',
          invalid: 'Indtast en gyldig IP-adresse',
          noDhcp: 'Ingen DHCP-DNS er tilgængelig i øjeblikket',
          saved: 'DNS-indstillinger gemt',
          saveFailed: 'DNS-indstillinger kunne ikke gemmes',
          unsaved: 'Ikke-gemte ændringer',
          maxServers: 'Maksimalt {{count}} DNS-servere er tilladt',
          dnsServers: 'DNS-servere',
          dhcpServersDescription: 'DNS-servere hentes automatisk fra DHCP',
          manualServersDescription: 'DNS-servere kan redigeres manuelt',
          networkDetails: 'Netværksdetaljer',
          interface: 'Grænseflade',
          ipAddress: 'IP-adresse',
          subnetMask: 'Undernetmaske',
          router: 'Router',
          none: 'Ingen'
        }
      },
      tailscale: {
        title: 'Tailscale',
        memory: {
          title: 'Hukommelsesoptimering',
          tip: "When memory usage exceeds the limit, garbage collection is performed more aggressively to attempt to free up memory. it's recommended to set to 50MB if using Tailscale. A Tailscale restart is required for the change to take effect."
        },
        swap: {
          title: 'Skift hukommelse',
          tip: 'Hvis problemerne fortsætter efter aktivering af hukommelsesoptimering, prøv at aktivere swap-hukommelse. Dette indstiller swap-filstørrelsen til 256MB som standard, som kan justeres i "Indstillinger > Enhed".'
        },
        restart: 'Are you sure to restart Tailscale?',
        stop: 'Are you sure to stop Tailscale?',
        stopDesc: 'Log out Tailscale and disable its automatic startup on boot.',
        loading: 'Indlæser...',
        notInstall: 'Tailscale ikke fundet! Installer det for at fuldføre opsætningen.',
        install: 'Installer',
        installing: 'Installerer',
        failed: 'Installation mislykkedes',
        retry: 'Opdater siden og prøv igen. Ellers prøv at installere manuelt.',
        download: 'Download',
        package: 'installationspakken',
        unzip: 'og udpak den',
        upTailscale: 'Upload tailscale til NanoKVM-mappen /usr/bin/',
        upTailscaled: 'Upload tailscaled til NanoKVM-mappen /usr/sbin/',
        refresh: 'Opdater sides',
        notRunning: 'Tailscale kører ikke. Start det for at fortsætte.',
        run: 'Start',
        notLogin:
          'Enheden er ikke tilknyttet en Tailscale-konto endnu. Log ind for at fuldføre tilknytningen til din konto.',
        urlPeriod: 'Denne URL er gyldig i 10 minutter',
        login: 'Log ind',
        loginSuccess: 'Log ind lykkedes',
        enable: 'Aktiver Tailscale',
        deviceName: 'Enhedens navn',
        deviceIP: 'Enhedens IP',
        account: 'Konto',
        logout: 'Log ud',
        logoutDesc: 'Er du sikker på, at du vil logge ud?',
        uninstall: 'Afinstaller Tailscale',
        uninstallDesc: 'Er du sikker på, at du vil afinstallere Tailscale?',
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
        loading: 'Indlæser...',
        notInstall: 'Ikke installeret',
        notConfigured: 'Ikke konfigureret',
        stopped: 'Stoppet',
        running: 'Kører',
        connected: 'Forbundet',
        error: 'Fejl',
        arguments: 'Argumenter',
        argumentsTip: 'Kommandolinjeargumenter, der sendes til tjenesten ved opstart.',
        env: 'Miljøvariabler',
        envKey: 'Navn',
        envValue: 'Værdi',
        envAdd: 'Tilføj variabel',
        envRemove: 'Fjern',
        configured: 'Konfigureret',
        save: 'Gem',
        saved: 'Konfigurationen er gemt',
        start: 'Start',
        stop: 'Stop',
        restart: 'Genstart',
        logs: 'Log',
        logsEmpty: 'Ingen logposter endnu',
        refresh: 'Opdater',
        binary: 'Binærfil',
        binaryShipped: 'Følger med firmwaren',
        binaryCustom: 'Egen binærfil',
        binaryUpload: 'Upload binærfil',
        binaryRevert: 'Gendan medfølgende binærfil',
        binaryRevertDesc: 'Slet den uploadede binærfil og gendan den, der følger med firmwaren?',
        serverWarning: 'En server uden begrænsninger fungerer som en åben proxy',
        noHealthSignal:
          'Denne tjeneste rapporterer ingen helbredsstatus, så NanoKVM ved kun, at processen kører, ikke om tunnelen er forbundet.',
        memoryWarning: 'Flere fjernadgangstjenester ad gangen kan opbruge hukommelsen',
        okBtn: 'Ja',
        cancelBtn: 'Nej'
      },
      update: {
        title: 'Kontroller for opdatering',
        queryFailed: 'Opdateringskontrol mislykkedes',
        updateFailed: 'Opdatering fejlede. Prøv igen.',
        isLatest: 'Du har allerede den nyeste version.',
        available: 'En opdatering er tilgængelig. Vil du installere den?',
        updating: 'Opdatering i gang. Vent venligst...',
        confirm: 'Bekræft',
        cancel: 'Annuller',
        preview: 'Forhåndsvisning af opdateringer',
        previewDesc: 'Få tidlig adgang til nye funktioner og forbedringer',
        previewTip:
          'Vær opmærksom på, at forhåndsvisningsudgivelser kan indeholde fejl eller ufuldstændig funktionalitet!',
        customServer: {
          title: 'Brugerdefineret opdateringsserver',
          desc: 'Søg efter og download onlineopdateringer fra en angivet server',
          invalidUrl:
            'Indtast en gyldig HTTP- eller HTTPS-servermappe uden forespørgsel, fragment eller latest.json.',
          loadFailed: 'Konfigurationen af opdateringsserveren kunne ikke indlæses.',
          saveFailed: 'Konfigurationen af opdateringsserveren kunne ikke gemmes.',
          saved: 'Konfigurationen af opdateringsserveren er gemt.',
          save: 'Gem',
          confirmTitle: 'Vil du bruge en brugerdefineret opdateringsserver?',
          confirmDesc:
            'SHA-512 kontrollerer kun, at pakken stemmer overens med manifestet fra denne server. Det beviser ikke, at pakken er en officiel NanoKVM-udgivelse. En fejlbehæftet eller ondsindet server kan gøre enheden ubrugelig, medføre tab af data eller kompromittere systemet.',
          confirm: 'Brug alligevel',
          previewDisabled:
            'Forhåndsvisningsopdateringer er ikke tilgængelige, mens en brugerdefineret opdateringsserver er aktiveret.'
        },
        offline: {
          title: 'Offline opdateringer',
          desc: 'Opdatering via lokal installationspakke',
          upload: 'Upload',
          checksumPlaceholder: 'SHA-256-kontrolsum (valgfri)',
          invalidChecksum: 'SHA-256-kontrolsummen skal indeholde 64 hexadecimale tegn.',
          checksumMismatch: 'SHA-256-verificeringen mislykkedes. Pakken kan være beskadiget.',
          invalidName: 'Ugyldigt filnavnsformat. Download venligst fra GitHub-udgivelser.',
          updateFailed: 'Opdatering fejlede. Prøv igen.'
        }
      },
      account: {
        title: 'Konto',
        webAccount: 'Navn på webkonto',
        password: 'Adgangskode',
        updateBtn: 'Update',
        logoutBtn: 'Log ud',
        logoutDesc: 'Er du sikker på, at du vil logge ud?',
        okBtn: 'Ja',
        cancelBtn: 'Annuller'
      }
    },
    picoclaw: {
      title: 'PicoClaw Assistent',
      empty: 'Åbn panelet og start en opgave for at begynde.',
      inputPlaceholder: 'Beskriv, hvad du vil have PicoClaw til at gøre',
      newConversation: 'Ny samtale',
      processing: 'Behandler...',
      agent: {
        defaultTitle: 'Generel assistent',
        defaultDescription: 'Generel hjælp til chat, søgning og arbejdsområde.',
        kvmTitle: 'Fjernstyring',
        kvmDescription: 'Betjen fjernværten gennem NanoKVM.',
        switched: 'Agentrolle skiftet',
        switchFailed: 'Kunne ikke skifte agentrolle'
      },
      send: 'Send',
      cancel: 'Annuller',
      status: {
        connecting: 'Opretter forbindelse til gateway...',
        connected: 'PicoClaw-session tilsluttet',
        disconnected: 'PicoClaw-session lukket',
        stopped: 'Stopanmodning sendt',
        runtimeStarted: 'PicoClaw runtime startet',
        runtimeStartFailed: 'Kunne ikke starte PicoClaw runtime',
        runtimeStopped: 'PicoClaw runtime stoppet',
        runtimeStopFailed: 'Kunne ikke stoppe PicoClaw runtime',
        controlSwitchedToMCP: 'Styringen er skiftet til den eksterne MCP-tjeneste'
      },
      connection: {
        runtime: {
          checking: 'Kontrol',
          restoring: 'Restoring PicoClaw',
          ready: 'Runtime klar',
          stopped: 'Runtime stoppet',
          blockedByMCP: 'Ekstern MCP-styring er aktiv',
          readyBlockedByMCP:
            'The runtime is running, but external MCP currently controls device input.',
          readyWithoutControl:
            'The runtime is running. Grant PicoClaw device control before reconnecting.',
          unavailable: 'Runtime utilgængelig',
          configError: 'Konfigurationsfejl'
        },
        transport: {
          connecting: 'Tilslutning',
          connected: 'Tilsluttet',
          disconnected: 'Disconnected',
          reconnect: 'Reconnect',
          reconnectDescription: 'Reconnect to the running PicoClaw session.',
          reconnectBlocked: 'PicoClaw needs device control before reconnecting.'
        },
        run: {
          idle: 'Tomgang',
          busy: 'Optaget'
        }
      },
      message: {
        toolAction: 'Handling',
        observation: 'Observation',
        screenshot: 'Skærmbillede'
      },
      overlay: {
        locked: 'PicoClaw styrer enheden. Manuel indtastning er sat på pause.'
      },
      control: {
        picoclaw: 'Enhedsstyring: PicoClaw',
        picoclawDescription: 'PicoClaw can write keyboard and mouse input. Manual input may pause.',
        mcp: 'Enhedsstyring: ekstern MCP',
        mcpDescription: 'External MCP can write to the device. PicoClaw will not take over input.',
        off: 'Enhedsstyring: fra',
        offDescription:
          'AI will not write keyboard or mouse input. Manual control remains available.',
        transitioning: 'Device control: switching',
        transitioningDescription: 'Device control is syncing. Please wait.',
        grant: 'Giv styring',
        release: 'Frigiv',
        releasing: 'Releasing...',
        switching: 'Switching...',
        releasingLabel: 'Device control: releasing',
        releasingDescription:
          'Device control is being returned. PicoClaw has stopped current writes.',
        granted: 'PicoClaw-styring givet',
        released: 'PicoClaw-styring frigivet',
        grantFailed: 'Kunne ikke give PicoClaw styring',
        releaseFailed: 'Kunne ikke frigive PicoClaw styring',
        grantConfirmTitle: 'Skift enhedsstyring til PicoClaw?',
        grantConfirmDesc: 'Eksterne MCP-enhedsskrivninger bliver afbrudt.'
      },
      install: {
        install: 'Installer PicoClaw',
        installing: 'Installation af PicoClaw',
        success: 'PicoClaw installeret korrekt',
        failed: 'Kunne ikke installere PicoClaw',
        uninstalling: 'Afinstallerer runtime...',
        uninstalled: 'Runtime blev afinstalleret.',
        uninstallFailed: 'Afinstallation mislykkedes.',
        requiredTitle: 'PicoClaw er ikke installeret',
        requiredDescription: 'Installer PicoClaw før start af PicoClaw runtime.',
        progressDescription: 'PicoClaw bliver downloadet og installeret.',
        stages: {
          preparing: 'Forberedelse',
          downloading: 'Downloader',
          extracting: 'Udpakning',
          verifying: 'Bekræfter',
          installing: 'Installerer',
          installed: 'Installeret',
          install_timeout: 'Timeout',
          install_failed: 'Mislykkedes'
        }
      },
      model: {
        requiredTitle: 'Modelkonfiguration er påkrævet',
        requiredDescription: 'Konfigurer PicoClaw-modellen, før du bruger PicoClaw chat.',
        docsTitle: 'Konfigurationsvejledning',
        docsDesc: 'Understøttede modeller og protokoller',
        menuLabel: 'Konfigurer model',
        modelIdentifier: 'Modelidentifikator',
        modelIdentifierPlaceholder: 'openai/gpt-5.4',
        apiBase: 'API Base URL',
        apiBasePlaceholder: 'https://api.example.com/v1',
        apiKey: 'API-nøgle',
        apiKeyPlaceholder: 'Indtast modellens API-nøgle',
        save: 'Gem',
        saving: 'Gemmer',
        saved: 'Modelkonfiguration gemt',
        saveFailed: 'Kunne ikke gemme modelkonfigurationen',
        invalid: 'Model-id, API Base URL og API-nøgle er påkrævet'
      },
      uninstall: {
        menuLabel: 'Afinstaller',
        confirmTitle: 'Afinstaller PicoClaw',
        confirmContent:
          'Er du sikker på, at du vil afinstallere PicoClaw? Dette vil slette den eksekverbare fil og alle konfigurationsfiler.',
        confirmOk: 'Afinstaller',
        confirmCancel: 'Annuller'
      },
      history: {
        title: 'Historik',
        loading: 'Indlæser sessioner...',
        emptyTitle: 'Ingen historik endnu',
        emptyDescription: 'Tidligere PicoClaw sessioner vil blive vist her.',
        loadFailed: 'Kunne ikke indlæse sessionshistorikken',
        deleteFailed: 'Kunne ikke slette session',
        deleteConfirmTitle: 'Slet session',
        deleteConfirmContent: 'Er du sikker på, at du vil slette "{{title}}"?',
        deleteConfirmOk: 'Slet',
        deleteConfirmCancel: 'Annuller',
        messageCount_one: '{{count}} besked',
        messageCount_other: '{{count}} beskeder',
        messageCount: '{{count}} beskeder'
      },
      config: {
        startRuntime: 'Start PicoClaw',
        stopRuntime: 'Stop PicoClaw'
      },
      start: {
        enableConfirmTitle: 'Skift styringen til PicoClaw?',
        enableConfirmDesc: 'Start af PicoClaw deaktiverer den eksterne MCP-tjeneste.',
        enableConfirmOk: 'Start PicoClaw',
        enableConfirmCancel: 'Annuller',
        title: 'Start PicoClaw',
        description: 'Start runtime for at begynde at bruge PicoClaw-assistenten.',
        switchFromMCP: 'Switch to PicoClaw and start',
        takeoverAndStart: 'Take over and start'
      }
    },
    error: {
      title: 'Vi er stødt på et problem',
      refresh: 'Opdater'
    },
    fullscreen: {
      toggle: 'Skift fuldskærm'
    },
    menu: {
      collapse: 'Skjul menu',
      expand: 'Udvid menu'
    }
  }
};

export default da;
