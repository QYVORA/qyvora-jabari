// Package banner provides the canonical JABARI brand banner (ASCII art).
//
// The art below is the exact byte-for-byte content of the repository-root
// file ascii_banner.txt, which is the single source of truth for the brand
// mark. Every surface of the tool (interactive console, terminal reports)
// MUST render this banner — never substitute custom art or hand-written
// wordmarks.
package banner

// Art is the JABARI brand mark: the robot / face symbol from the product
// logo. The glyphs follow the brand palette — '%' is the lime-green
// (#85C236) foreground of the robot body, '#' is the deep blue-black
// (#19222B) circular background, and '+' / '*' are the white (#FFFFFF)
// structural details (antenna, signal waves, eyes).
const Art = `                        %%%%%%%%%%%%%%                          
                  %%%%%%%%%%%%%%%%%%%%%%%%%                 
              %%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%             
           %%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%          
        %%%%%%%%%%%%%%%%%%%%%#+*%%%%%%%%%%%%%%%%%%%%%       
      %%%%%%%%#%%%%%%%%%%%%%*++++%%%%%%%%%%%%##%%%%%%%%     
     %%%%%%%%%+++%%%%%%%%%%*++++++*%%%%%%%%%+++%%%%%%%%%    
    %%%%%%%%%%%*++%%%%%%%%###+++####%%%%%%%++%%%%%%%%%%%%%  
  %%%%%%%%%%%%%%%#++%%%*++++++++%%%++#%%%++#%%%%%%%%%%%%%%% 
  %%%%%%%%%%%%%%%%*+++++++++++++%%%%%*++++#%%%%%%%%%%%%%%%% 
 %%%%%%%%%%%%%%#*+++++++++++++++%%%%%%%%%++#%%#%%%%%%%%%%%%%
%%%%%%%%%%%%%*++++++++++++++++++%%%%%%%%%%%%%%%%+++%%%%%%%%%
%%%%%%%%%%%#++++++++++++++#*++++%%%%%%%%%%%%*++#%%*+#%%%%%%%
%%%%%%%%%%#++++++++++++++%%%#+++%%%%%%%%%%%#%%%++*%#++%%%%%%
%%%%%%%%%%++++++++++++++++%#++++%%%%%%%%%%%#++#%*+*%#+*%%%%%
 %%%%%%%%++++++++++   ++++%%++++%%%%%%   %%%%*+*%%+#%#+%%%%%
  %%%%%%#++++++++++   +++++*%#++%%%%%%   %%%%#+*%%+*%#+#%%% 
  %%%%%%*++++++++++++++++++++%*+%%%%%%%%%%%%%%#%%%+#%#+%%%% 
    %%%%#++++++++++++++++++++%*+%%%%%%%%%%%%%%%%%%%%%#%%%   
     %%%%%%**+++++++++++++++%#+++%%%%%%%%%%%%%%%%%%%%%%%    
       %%%%%%%%%%%%%%%%%%%%%*++++++%%%%%%%%%%%%%%%%%%%%     
        %%%%%%%%%%%%%%%%%%%%%+++++#%%%%%%%%%%%%%%%%%%       
           %%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%           
              %%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%%             
                   %%%%%%%%%%%%%%%%%%%%%%%%
`
